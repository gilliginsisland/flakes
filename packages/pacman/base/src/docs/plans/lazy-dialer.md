# Lazy dialer lifecycle and race reference

Status: reference for the implemented code, replacing the earlier design draft.
Read this alongside `pkg/dialer/lazy.go`; it describes current behavior rather
than a proposed rewrite. Update both when changing the protocol below.

PACman creates shared tunnels on demand. One Lazy caches a factory result and
serves many connections. Its atomic ready bit admits readers of the cached
fields; its monitor serializes factory publication, retirement, releases, idle
timing, and state updates.

## Ownership and behavior

- A factory returns `Dialer, error`. Dialer implements `Dial`, `DialContext`,
  `Close`, and `Wait`. A successful factory returns a non-nil dialer. On error,
  the factory owns cleanup and returns nil; Lazy does not dispose of partial
  results returned alongside an error.
- Request contexts cover individual dial attempts. They are not connection
  reference counters and must not own an established connection's lifetime.
- Each successful acquisition holds one reference through dialing and through
  the returned connection's lifetime. Dial failure releases it; otherwise the
  caller must Close the returned connection, including after EOF.
- Backend Close must interrupt its pending dials and close its established
  connections, and Wait must eventually return. Close may be called after Wait
  or more than once. Lazy does not add a context-cancellation callback to every
  returned connection.
- Factory errors remain cached until the Reset action clears them. Natural
  tunnel death and idle retirement clear the successful result after Wait and
  reference draining, allowing a later request to reconnect.
- `Reset(action)` queues one of four intent-specific commands. It is not a
  general disconnect-and-clear operation. Permanent Close is separate.
- Lazy publishes state snapshots. The pool owns menus and notifications; it
  does not manage the underlying tunnel or maintain a second lifetime context.

## State and field ownership

Lazy has no initialization mutex, generation pointer, slot array, or separate
snapshot-reader counter. Its packed `atomic.Uint64` is:

```text
bit 63                    bits 62 through 0
lazyReadyBit              aggregate reference count
1 bit                     63 bits
```

`state.Add(1)` returns the new word and reserves one reference while observing
readiness in the same atomic operation. Ready means the cached `value` and `err`
are published, not that the network is healthy. A cached error also has ready set.

Only the monitor writes `value` and `err`. Callers read them only while holding
a reference whose Add observed ready. That reference protects both the field
copy and subsequent use; there is no separate notification for finishing a copy.

The count includes active dials, returned connections not yet Closed, ready
releases queued but not consumed, and brief not-ready acquisitions between Add
and release. Callers parked on the request channel hold no reference.

The factory, pending update, publication `rundown`, timer, and channel enable
flags belong to the monitor and its helpers. Workers send results; they do not
clear or publish Lazy's cached fields. The lifetime context and cancel function
are created once by NewLazy and are never replaced.

Counter arithmetic assumes fewer than `2^63` outstanding references and exactly
one release per acquisition. There is no overflow guard. Do not reset the packed
word wholesale while callers can still release references.

## Channels and helpers

| Mechanism | Purpose |
| --- | --- |
| `requests` | Unbuffered offers sent by the monitor; a waiting caller receives an offer and retries acquisition. No result or per-request reply channel is carried. |
| `releases` | A 256-entry channel carrying the ready bit observed at acquisition: either zero or `lazyReadyBit`. These are tags, not arithmetic deltas. |
| `resets` | A one-entry queue of ResetAction commands. Different actions are not coalesced. |
| `wait` | One stable, one-entry error channel. Each successful factory starts one goroutine sending its backend Wait result. |
| `factory` | A Deferred holding the single active factory's result and setup cancellation. A nil Deferred has a nil Done channel. |
| `timeout` | The reusable timer's channel, or nil when idle timeout is disabled. |
| `updates` | A pending send of the latest StateUpdate, without blocking lifecycle work. |

Switchable stores the original channel in `Value()`. Its `Current()` returns
that channel when enabled and nil when disabled. Only the monitor's request
send uses `requests.Current()`; callers always receive from `requests.Value()`.
This lets enabling offers wake callers that were already waiting. Reset senders
and Wait workers likewise use Value so pausing a monitor receive does not change
the channel they use. Switchable does not interrupt a select already in progress.

Deferred starts the factory in a goroutine with a child of Lazy's lifetime
context. Done closes only after the factory returns and its values are written.
Cancellation requests do not close Done early. Deferred cancels the setup
context on return; returned resources must own an independent lifetime.

## Acquisition, dialing, and release

`acquire(ctx)` returns `(Dialer, func(), error)`:

1. Add one to state.
2. If the returned word is ready, copy `value` and `err`. For an error, release
   the ready reference immediately and return the cached error. For success,
   return the dialer and a release function guarded by `sync.OnceFunc`.
3. If not ready, call `release(0)` before waiting. Select between a monitor offer,
   the caller's context, and Lazy's lifetime context. An offer means retry from
   step 1; it does not authorize an uncounted read of the cached fields.

The first offer while empty starts the shared factory. That caller may retry
before setup completes, observe not ready again, and park again. Requests are
disabled during setup, then enabled on publication so waiters can retry. When
already ready, an offer only lets an existing slow waiter retry; it starts no
additional factory.

The fast path does not separately check either context. Caller cancellation is
handled by the backend DialContext, while lifetime Close asks the monitor to
close the backend. A race can still admit an acquisition before retirement is
processed. Do not document acquisition as a synchronous closed-state check.

Lazy.DialContext calls the acquired dialer with the caller's context. A non-nil
error releases the reference without inspecting or closing a partial connection.
On success it returns trackedConn. trackedConn calls the underlying Close and
then its release hook, even if Close reports an error. Repeated Close calls can
reach the underlying connection, but the Lazy reference is released only once.

Release follows this protocol:

| Acquisition observed | Release behavior |
| --- | --- |
| Not ready, decrement still observes not ready | Subtract one atomically and return without a channel message. |
| Not ready, decrement observes ready | Subtract one atomically, then send zero to acknowledge publication. |
| Ready | Send `lazyReadyBit`; the monitor performs the decrement. |

A ready release stays counted until the monitor consumes it. This ordering is
required for idle timing. The send may block if the buffer is full. Both kinds
of send can abandon notification when Lazy's lifetime context is canceled;
counts no longer need to balance after permanent retirement.

## Publication and rundown

The monitor reads the completed Deferred, starts a Wait sender for a successful
result, and stores `value` and `err`. It then publishes with:

```go
rundown = state.Or(lazyReadyBit) &^ lazyReadyBit
```

Or returns the old word. Before publication there were no ready readers, so this
snapshot counts only not-ready acquisitions still between their Add and their
atomic release. It does not count callers already parked on requests.

If the snapshot is nonzero, the monitor disables reset, Wait, and timeout
receives. Requests remain enabled and ready callers continue using the result.
Each zero-tagged release reduces rundown; ready-tagged releases decrement state
but do not reduce rundown. When rundown reaches zero, those lifecycle channels
are enabled again.

This barrier prevents a not-ready observer from spanning two readiness cycles.
A very short-lived tunnel may already have died during rundown; its Wait result
can sit in the buffer, but ready remains set until all publication acknowledgments
arrive. New callers may therefore briefly acquire that dead backend and fail.
That is an intentional tradeoff, not a claim that ready guarantees health.

## Retirement and replacement

`retire()` disables request offers and atomically clears ready with And. If ready
was set, the old count is the exact number of ready references to drain, because
publication rundown has already completed before retirement can be handled.

For a successful result it starts backend Close in a goroutine. The monitor then
uses `syncutil.Discard` to consume that many release messages and subtracts the
number received in one atomic operation. It drains whole connection references,
not just short field readers. New callers seeing not ready add and subtract
their own transient references directly; they do not add messages to this drain.

Discard is the same monitor consuming releases, not a second consumer. It can
stop early on permanent cancellation, in which case retire returns false and
the monitor exits. Otherwise retire stops the idle timer and returns true.

During this inner drain, ordinary monitor events are not processed. Backend
Close and Wait must not depend on the monitor servicing those other events,
and owners must Close returned connections after backend shutdown wakes their
I/O. A retained wrapper that is never Closed can prevent retirement indefinitely.

After Disconnect, the old `value` remains installed with ready cleared and
requests disabled until its Wait result is consumed. The Wait case completes
retirement, clears `value` and `err`, enables requests, and publishes Offline.
Natural death takes that same Wait case. Cached-error Reset has no Wait sender:
it drains error readers, clears `err`, enables requests, and publishes Offline.

Only one successful backend is observed at a time. Its Wait result is consumed
before a replacement factory starts, so a late result cannot clear a replacement.
The one-slot Wait buffer also allows its sender to finish after monitor exit.

## Idle timing

NewLazy constructs and immediately stops one timer when IdleTimeout is nonzero.
Zero disables idle timeout; other durations follow time.Timer semantics.

`resetIdleTimer()` resets the full interval only when all of these hold:

- A timer exists.
- State is exactly `lazyReadyBit`: ready with zero references.
- The cached error is nil.

The helper runs after publication and after each normally processed release.
Publication must handle zero references itself: shared setup can outlive the
request that started it, and Connect can start it with no caller at all.

A timeout claims an unused backend only with `CompareAndSwap(lazyReadyBit, 0)`.
If that fails, it does nothing. If it succeeds, the monitor disables requests,
starts backend Close asynchronously, and waits through its normal select for
the existing Wait sender. The timer is one-shot; consuming a tick does not
require setting its channel to nil. Cached factory errors never arm this timer.

An acquisition need not stop the timer. Its Add makes the idle CAS fail. Ready
releases must remain queued until the monitor can both subtract them and reset
the timer in the same select case. A direct decrement followed by a reset would
let an already-dispatched timeout claim the backend too early.

This ordering relies on the modern timer-channel behavior used by the module's
Go version. Do not apply the same stale-tick argument when explicitly opting
into legacy behavior through `GODEBUG=asynctimerchan=1`.
See [Timer.Reset](https://pkg.go.dev/time#Timer.Reset) and
[Timer.Stop](https://pkg.go.dev/time#Timer.Stop).

## Actions and permanent Close

Reset(action) sends to the one-entry queue or returns when Lazy is permanently
canceled. It can block while the queue is full or action handling is paused.
It has no completion acknowledgment. The monitor checks applicability when it
handles the command, not when the user clicked the menu.

| Action | Applicable state | Effect |
| --- | --- | --- |
| Cancel | Factory exists and Done is not yet closed | Cancel its setup context with ErrResetRequested. Await and publish the factory's actual result. |
| Disconnect | `value != nil` | Retire the backend; remain unavailable until Wait completes. |
| Reset | `err != nil` | Retire cached error readers, clear the error, and publish Offline. |
| Connect | No factory, value, or error | Start shared setup without requiring a network request. |

Other combinations are ignored. A queued Cancel cannot turn into Disconnect,
and a queued Reset cannot cancel setup or close a successful tunnel. Commands
have no generation identity: an applicable command acts on the current result.

Setup cancellation is not given a special publication policy. If the factory
returns an error after Cancel, the error is cached and Failed is published. If
it nevertheless returns success, that success is published. Lazy does not
convert a successful return into an error by checking the setup context later.

Close permanently cancels the lifetime context with ErrCloseRequested. Done
closes on that request, not on monitor exit or completion of backend cleanup.
Once the monitor selects cancellation it:

- Arranges for an outstanding factory result to be Closed if it succeeds after
  the monitor leaves. It does not start a Wait observer for that discarded result.
- Clears ready, closes a published backend if it owned the ready result, and
  returns without clearing fields or draining all references. If retirement
  already cleared ready, its asynchronous Close is already responsible for it.

Keeping the fields unchanged on shutdown lets already-admitted readers finish
copying them safely. Their backend is being closed; no replacement is published.
A canceled select can still choose a simultaneously ready offer or result first,
so Close is not a synchronous barrier against every last admission or publication.
The lifetime context remains canceled and the result is eventually disposed.

## Pool, notifications, and backend lifetimes

The pool supplies a factory with a label and a two-minute setup timeout. It
cancels that timeout on factory return. It does not use that context to keep a
successful tunnel alive or retain its connections. Deferred also cancels its
own setup child after completion.

The pool subscribes to Lazy's state updates and updates its existing menu items:

| Update | Meaning |
| --- | --- |
| Connecting | Shared setup started. |
| Online | The factory returned success; stateless dialers may not have contacted their upstream yet. |
| Failed | A factory error was cached. |
| Offline | The previous backend ended, or a cached error was cleared. Wait errors are attached to Offline, not cached as factory failures. |

Construction does not emit an initial Offline update; the pool initializes its
menu to Offline itself. Delivery is a latest-state mechanism: notify tries an
immediate send and otherwise retains one pending snapshot. A newer update can
replace that pending snapshot, so intermediate notifications are not guaranteed.
The monitor retries the latest pending update while alive; shutdown can end it
without delivering that pending update. The pool does not infer state from its
factory being called or from observing backend Wait independently.
With a nil Updates channel, state delivery is disabled.

Each menu action has its own fixed tag and visibility. The callback sends the
corresponding ResetAction, preserving the meaning of a queued click across state
changes. Configuration reload reuses unchanged pooled dialers; replacement rules
and menus are installed before removed or changed dialers are Closed.

For a backend lacking the full Dialer interface, forwardDialer extracts its
native DialContext, Close, and Wait capabilities once. Native Close owns its
connections. Without it, the shim maintains a mutex-protected connection map
and a separate lifetime context, and Close cancels pending setup and closes
registered connections. Without native Wait, the shim supplies a completion
channel closed by its own Close. It does not infer tunnel death from one failed
connection. These internal lifetime signals are separate from request contexts.
For a backend without DialContext, the fallback can stop waiting for a blocking
Dial but cannot stop its goroutine. It closes any late connection when Dial
eventually returns; shim Wait does not join that underlying background call.

OpenConnect likewise owns a separate lifetime after authentication/setup. Its
native loop runs from a Go goroutine. Once started, the loop must return before
connection cancellation schedules native cleanup; Close before Run schedules
cleanup directly. TunClient borrows a raw packet fd into OpenConnect and owns
both socket endpoints in Go, closing them from a goroutine waiting on Conn.Wait.
Run starts the native loop before installing the packet fd. The synchronous
setup-tunnel callback installs it and constructs netstack after initial MTU
detection. A callback failure returns a negative errno to native startup, and
MainLoop returns the original Go error through the callback error channel.
The factory waits for setup and detaches its cancellation hook before returning
the independently owned tunnel. Conn.Wait reports that native work stopped;
the command pipe, VPN object, and socket cleanup can finish later.
Lazy does not require all asynchronous resource cleanup to finish before the
backend's Wait returns.

## Race catalogue

Each pair names both orderings and the boundary that makes them acceptable.
The liveness assumptions after this catalogue remain part of the contract.

### 1. Concurrent slow requests

- A receives the first offer: the monitor starts one factory and disables offers;
  A and B then wait or retry until publication. Neither can start its own factory.
- B receives first: the same sequence with roles reversed. The monitor, not the
  first atomic Add, owns setup.

### 2. Acquisition versus publication

- Add happens before Or: the caller observes not ready, does not read the fields,
  and releases before waiting. If still counted at Or it owes one zero message.
- Or happens first: Add observes ready and protects the completed field values
  until its ready release is consumed. Publication precedes those field reads.

### 3. Not-ready release versus publication

- The decrement happens first: its returned state is not ready, no message is
  sent, and the later publication snapshot excludes it.
- Or happens first: the decrement sees ready, sends zero, and the snapshot
  includes exactly that acknowledgment. Processing it reduces rundown once.

### 4. Immediate death versus publication rundown

- Wait finishes before rundown: its result buffers while Wait handling is
  disabled. Acquisitions can reach the dead backend, but fields are not replaced.
- Rundown finishes first: Wait handling is enabled and can immediately retire
  the backend. No not-ready observer spans the next readiness cycle.

### 5. Acquisition versus retirement

- Add observes ready before And clears it: the snapshot includes that reference;
  retirement cannot replace fields until its release has been drained.
- And happens first: the caller observes not ready and releases directly before
  waiting. It cannot read the old fields or extend the captured drain target.

### 6. Ready release versus retirement

- The monitor consumes the release first: state is decremented and the later
  snapshot excludes it.
- Retirement snapshots first: that reference remains counted, whether its
  message is already queued or sent later. Discard consumes it and the batch
  subtraction removes it exactly once.

### 7. New not-ready arrivals during a drain

- Their Add and direct decrement finish before the batch subtraction: they
  cancel out; only the old ready references are subtracted by retire.
- A transient Add is still outstanding when the batch is subtracted: its count
  is preserved. Its later direct decrement balances it. New arrivals send no
  messages while ready is clear, so they cannot inflate the old drain target.

### 8. Acquisition versus idle CAS

- Add wins: the word contains a reference, so CAS of exactly ready to zero fails.
- CAS wins: the caller sees not ready and waits for the old backend to end. It
  never reads the result that timeout claimed.

### 9. Last release versus an expired timer

- Timeout is handled first: an unconsumed ready release still counts, so CAS
  fails. Handling the release later resets the full interval.
- Release is handled first: subtraction and reset happen in one monitor case.
  Modern timer semantics prevent the old unread tick from winning afterward.
  The same argument covers a very short successful or failed dial attempt.

### 10. Setup completion versus the initiating request leaving

- The request releases before publication: the snapshot can be zero, and
  publication itself arms idle timing for a successful unused result.
- Publication catches its transient reference: the zero acknowledgment ends
  rundown; release processing checks whether the now-unused result needs a timer.
  A factory started by Connect is covered by the zero-snapshot path as well.

### 11. Cancel versus factory completion

- The monitor sees Done already closed: Cancel does nothing; the actual result
  is published normally.
- The monitor cancels first: the factory is asked to stop. Its returned error
  becomes Failed and remains cached, or its returned success becomes Online.
  A context cancellation after success does not rewrite that result.

### 12. Delayed actions versus state changes

- The action is handled while applicable: only its named operation runs.
- State changes first and makes it inapplicable: it is ignored. Cancel never
  becomes Disconnect, Reset never becomes Cancel, and Connect cannot bypass a
  cached error. Applicability is about current state, not a captured generation.

### 13. Cached-error Reset versus acquisition

- Add observes ready first: the caller copies the error and releases; Reset
  drains its reference before clearing the cached fields.
- Reset clears ready first: the caller releases without reading and waits for
  requests to be enabled. It retries and can trigger fresh setup.

### 14. Wait completion versus timeout or Disconnect

- Wait is handled first: retire clears ready, drains references, and stops the
  timer before permitting another factory.
- Timeout or Disconnect starts closing first: requests stay disabled and the
  old value remains installed until its Wait result is consumed. The Wait case
  does not launch a replacement ahead of the old backend. A stopped old timer
  cannot later claim the replacement under the timer semantics above.

### 15. Caller cancellation versus a request offer or successful dial

- The slow-path select chooses cancellation: it returns the caller's error with
  no reference retained. A shared factory already triggered may still complete.
- It chooses an offer: acquisition retries. If the backend returns success, the
  caller owns Close; Lazy adds no post-success context check or automatic close.
  If the backend returns an error, Lazy releases the reference. The backend is
  responsible for honoring setup cancellation and cleaning up failed operations.

### 16. Repeated connection Close

- The first Close finishes: its once-only hook queues the reference release.
- Another Close finishes first or later: sync.OnceFunc still runs the release
  only once. Underlying net.Conn.Close must support its normal repeated and
  concurrent use; the wrapper does not serialize the native Close itself.

### 17. Permanent Close versus factory publication

- The monitor selects lifetime cancellation first: a separate waiter takes and
  closes any eventual successful factory result; no result is abandoned.
- It selects the completed factory first: it may publish, then the cancellation
  case clears ready and closes that backend. Both paths dispose of a success;
  neither makes the lifetime context usable again.

### 18. Permanent Close versus release or drain

- The release is handled before cancellation: ordinary accounting and timer
  work complete; retirement can then leave through its cancellation path.
- Cancellation wins first: release sends and Discard can stop without balancing
  all counts. The fields are not overwritten and no replacement is created after
  monitor exit, so abandoned counts cannot affect another generation.

### 19. State update versus receiver availability

- The receiver accepts the snapshot: it observes the monitor's current update.
- The receiver is busy: a later state can replace the pending snapshot. Only
  delivered states drive menu updates and notifications; no old observer can
  asynchronously overwrite a newer monitor state.

### 20. Permanent Close versus an action

- The action is processed first: it can start or retire setup according to its
  usual applicability, but later lifetime cancellation still disposes of the
  result. It cannot make the canceled lifetime usable again.
- The monitor handles cancellation first: it exits without processing queued
  actions. Reset senders can stop waiting through the lifetime context. A send
  racing with cancellation can still enqueue an action, but no new factory is
  started by that message after monitor exit.

## Assumptions and maintenance checks

- Backend Close must make its dials and connection I/O finish, and owners must
  eventually Close returned connections. Draining can wait indefinitely when
  either condition fails. A finite channel buffer is not a liveness proof.
- Factory code must honor cancellation or eventually return. It must not
  recursively acquire the same Lazy through a proxy cycle: its own requests
  are disabled while the monitor waits for that factory.
- Publication rundown must finish before enabling any event that clears ready.
  Keep zero acknowledgments separate from ready-reference decrements.
- Only the monitor consumes releases, including its inner Discard. Do not add
  a second consumer or replace ready releases with direct decrements without
  re-establishing both the publication and timer proofs.
- Reference and publication-ack producers can wait on the 256-entry release
  buffer. Do not make backend Close depend on the monitor handling unrelated
  messages while retire is draining.
- Done reports permanent Close being requested, not completed teardown. Backend
  Wait and UI delivery have their own weaker completion boundaries described
  above. Do not infer that every resource has been freed from a state update.
- This document records source reasoning, not a runtime race-freedom proof. For
  code changes, use the project's build and vet checks and report any live
  verification separately. Do not add or run tests without explicit permission.

## Source map

Paths are relative to the Go module at `packages/pacman/base/src`.

| File | Responsibility |
| --- | --- |
| `pkg/dialer/lazy.go` | Atomic admission, publication rundown, releases, actions, timer, retirement, and state updates. |
| `pkg/dialer/common.go` | trackedConn release hook and fallback context-aware dialing. |
| `pkg/dialer/connectionstate.go` | State and update types. |
| `pkg/syncutil/deferred.go` | Factory setup context, completion, and values. |
| `pkg/syncutil/switchable.go` | Channel enable/disable behavior and stored Value access. |
| `pkg/syncutil/channel.go` | Counted, cancellable Discard used by retirement. |
| `internal/app/pool.go` | Factory label/timeout, menu state, and notification consumption. |
| `internal/app/systray.go` | Fixed menu action identity and ResetAction dispatch. |
| `internal/app/app.go` | Pool reuse and configuration replacement ordering. |
| `pkg/dialer/dailer.go` | Native lifecycle capability detection and the forwarding shim. |
| `pkg/dialer/oc/oc.go` and `pkg/openconnect/client.go` | OpenConnect setup, native-loop lifetime, and Go-owned fd cleanup. |
| `pkg/httpproxy/server.go` and `pkg/sshproxy/server.go` | Request contexts and explicit proxied connection closure. |
