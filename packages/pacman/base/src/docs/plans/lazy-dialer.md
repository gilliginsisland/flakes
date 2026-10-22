# Lazy dialer design and race plan

Status: design draft for review before another implementation pass.

PACman lazily creates shared tunnels for configured proxies. This design keeps
the incoming-request path small, makes ownership explicit, and treats cleanup
as a protocol rather than a collection of independent callbacks. Goals come
first; the candidate design below is a way to meet them, not the definition of
success. Its race catalogue lists both orderings at each concurrency boundary.
Unproved liveness cases are marked explicitly; they are not declared safe merely
because the code compiles.

This is a planning change only. No implementation or tests are part of this step.

## Goals

- A tunnel is shared by many proxied connections. Construct it on demand, retain
  it while it is used, and close it after the configured idle duration.
- Concurrent requests share one factory result rather than starting duplicate
  tunnels. A completed result can be a dialer or a cached error.
- Keep the common acquisition path fast under concurrent requests. Cleanup and
  failure handling must not make every request contend on an initialization lock.
- Reset disconnects the current tunnel and clears the cached result, allowing
  later requests to reconnect. Reset must not leave an obsolete result usable.
- Close permanently retires the lazy dialer. A closed lazy dialer cannot be reset
  into service. Pool removal uses Close.
- The caller's context controls its dial attempt, not its connection reference.
  Once a connection is returned, its Close releases that reference.
- Once tunnel death is observed and the result is invalidated, new requests must
  not reuse it. Existing users must remain safe while replacement takes place.
- Timeout, Reset, death, and connection closure must not lose references, close
  a replacement accidentally, or leave requests blocked indefinitely by teardown.
- UI state and notifications belong to the pool's factory and dialer wrapper.
  Preserve their lifecycle ordering without putting menu logic in Lazy.

## Design preferences

These are the mechanisms requested for the design, separate from its behavioral
goals. Their correctness and costs still need to be checked against those goals.

- Keep a stable lazy dialer per configured proxy so rules can retain it across
  Reset without being rebuilt.
- Pack readiness, snapshot readers, and connection references into one 64-bit
  word, with a single Add to acquire both counts on the common path.
- Send decrements through one 256-slot channel so one loop owns decrement and
  timeout ordering.
- Keep all counter operations on Lazy, not on a separate counter object.

## Candidate design and tradeoffs

The remaining layout, synchronization, and race outcomes describe the current
candidate. They are subject to review, not additional goals.

In this candidate, a factory runs while holding mu, so Reset waits for setup
before clearing its result. That avoids separately synchronizing access to a
factory's cancellation function, but Reset can be delayed by a slow factory.
Cancelling setup promptly is an alternative to evaluate, not something ruled
out by the goals. We must decide its required responsiveness and publication
ordering before selecting the mechanism.

The existing pool-owned Disconnect can cancel setup immediately using its own
synchronization. This is an existing capability, not a reason to define waiting
as a Reset requirement. Any additional synchronization in Lazy should solve a
specific requirement identified during this review.

The candidate depends on these factory and dialer properties: setup must
eventually finish or honor lifetime cancellation; Dial, Close, and Wait must support their actual
concurrent use; successful Close must eventually let Wait finish. Recursive
forwarding may use other lazy dialers, but a factory must not dial itself through
a direct or indirect cycle while holding its own mu. No local refcount algorithm
can make that lock cycle safe.

## Names and proposed result shape

Use one result value to keep the dialer, its error, and its generation context
together:

```go
type lazyEntry struct {
	dialer Dialer
	err    error
	ctx    context.Context
}

func (d *Lazy) acquire(ctx context.Context) (lazyEntry, func())
```

The entry is returned by value. Its interfaces refer to the old generation even
after Lazy replaces its stored entry. The release function belongs to this
acquisition, not to the cached entry. It is nil on an error result and guarded
by sync.OnceFunc on success.

Keep the generation cancel function and Wait-completion signal as control data
protected by mu. Do not return them to each request. Names, ownership, and
sequencing below describe the candidate and must be checked against the goals.

The factory result still implements dialing, Close, and Wait. A successful
factory must return a non-nil dialer with a nil error. A non-nil partial dialer
returned with an error must be closed rather than published as a usable tunnel.

## Two contexts with different lifetimes

The lifetime context and cancel function are created once in NewLazy and never
reassigned. Close cancels this context without taking the initialization mutex.
That cancellation reaches factories and connection attempts through their child
contexts. No separate cancelMu is needed for this immutable cancel function.

Each initialization creates a fresh generation context derived from the lifetime
context. The factory receives it. The entry retains it so a request can combine
its own setup cancellation with cancellation of the exact generation it acquired.

Generation cancellation also closes returned connection wrappers. This matters
for forwarding dialers whose Close does not itself own every connection they
returned. Reset and idle cleanup cancel the generation without cancelling the
lifetime context. Tunnel Wait completion also ends that generation.

The pool may derive a further context for labels, setup timeout, and its existing
Disconnect operation. These contexts do not own connection references. After a
successful dial, cancelling the request context must not release the reference
or automatically close the established connection.

## The packed 64 bit state

Use one atomic 64-bit word. The current implementation uses atomic.Uint64;
the logical layout does not depend on treating the word as a signed number.
There are two counts, but reserving the high bit means they cannot both be full
32-bit counts.

```text
bit 63             bits 62 through 32       bits 31 through 0
ready              snapshot readers        connection references
1 bit              31 bits                 32 bits
```

```go
readyBit   = uint64(1) << 63
readerUnit = int64(1) << 32
refMask    = uint64(readerUnit - 1)
readerMask = (readyBit - 1) & ^refMask
acquireAdd = uint64(readerUnit + 1)
```

An acquisition performs one state.Add(acquireAdd). The returned word both
reserves a connection reference and tells the caller whether it may copy the
published fields without taking mu. It also reserves one snapshot-reader count.

All subtraction is performed by the cleanup goroutine. Producers send positive
int64 deltas to a 256-slot decrement channel:

| Delta | Meaning |
| --- | --- |
| readerUnit | Finished copying, or observed not ready and will not copy |
| 1 | Release a connection reservation or connection reference |
| readerUnit + 1 | Cached error: release the reader and reference together |

The consumer applies state.Add(uint64(-delta)). No producer performs that Add.
No zero-wakeup messages or direct reader decrements are needed in this design.

The counts include releases that have been sent but not yet consumed. Reference
counts include requests waiting on mu, in-progress dials, returned connections,
and old-generation releases not yet processed. They are not just a count of
currently open sockets.

Arithmetic assumes neither count can overflow its allocated field and that
every decrement has a matching acquisition. A low-count carry would corrupt
the reader count; a reader-count carry would corrupt readiness. Do not present
the packing as correct without those bounds. Do not zero the entire word on
Reset or tunnel death: old releases would subtract from the replacement's counts.

## Candidate ownership and invariants

1. Initializers and cleanup writers hold mu whenever they modify entry or
   generation control fields. The fields become readable only after complete
   atomic publication of readiness.
2. A fast reader whose Add observes ready may copy entry. It queues its reader
   decrement only after that copy is finished and does not subsequently read the
   shared entry. Continued use is through its local snapshot.
3. A caller whose Add observes not ready queues its reader decrement before
   waiting on mu. It does not read the shared entry outside mu.
4. Clearing readiness blocks new unprotected copies; it does not prove that old
   copies are finished. Cleanup holds mu and drains the reader count before
   replacing any shared field.
5. New not-ready acquisitions still temporarily add a reader reservation. The
   drain can conservatively include them until their queued decrements are
   applied. This is not RWMutex's separate admitted-reader accounting.
6. Acquiring mu after that drain is the handoff that makes replacement
   initialization safe. Initialization does not perform a second drain.
7. Connection references survive snapshot completion. Exactly one reference
   release follows acquisition failure, dial failure, or connection Close.
8. Only the cleanup goroutine consumes decrements, modifies the idle timer, and
   performs Reset, death, timeout, and permanent-retirement cleanup. Draining
   inside its cleanup helper is still the same consumer.
9. An idle claim requires the same generation and a CAS of exactly readyBit to
   zero while holding mu. Any reader or connection count prevents that claim.
10. Reset and death may clear readiness regardless of connection references.
    They drain snapshot readers, not the lifetime of all connections.
11. Observe the old dialer's Wait wrapper finishing before permitting another
    factory. Its Offline update must be queued before replacement Connecting.
12. Permanent cancellation prevents new factories. Counts and late releases can
    delay worker retirement, but cannot revive the lifetime context.

## Acquisition and dialing

### Ready fast path

1. Reject a cancelled request or retired lazy dialer.
2. Add one reader and one reference in a single atomic operation.
3. If ready is set, copy entry by value.
4. If entry.err is non-nil, queue readerUnit + 1 and return the error snapshot
   with no release function.
5. Otherwise queue readerUnit and return the snapshot and a once-only release
   function that queues 1.

There is no initialization mutex or CAS retry loop on this path. The channel
send can apply backpressure; a buffer is not a guarantee that it never blocks.

### Not ready slow path

1. Queue readerUnit before trying to acquire mu. Retain the connection reference.
2. Acquire mu and recheck permanent cancellation, caller cancellation, and ready.
3. If still not ready, initialize exactly once under mu. Other callers do not
   independently infer that they are the initializer from an unset ready bit.
4. Copy the completed entry while holding mu, then unlock.
5. On failure queue 1. On success return the snapshot and its once-only release.

Waiting on mu is not promptly cancellable in this candidate. A caller cancelled
during that wait unwinds when it gets the lock. This is a cancellation-latency
tradeoff to evaluate, not a behavioral goal.

### Dial and returned connection

Combine caller cancellation during setup with generation cancellation, then
invoke the captured dialer. On failure, close any partial returned connection
and release the reference. Audit the completion check so a cancelled setup that
returns a connection cannot leak it.

On success return a wrapper that closes the underlying connection and queues the
once-only release. Install the generation-cancellation callback before exposing
that wrapper. Disable the setup-only cancellation link after dialing. The caller
owns connection Close, including after EOF. EOF alone is not a reference release.

Repeated Close and a concurrent cancellation callback must release only once.
If the underlying dialer or connection is not safely closeable more than once,
the wrapper must also serialize the underlying Close, not just its counter release.

## Initialization and result publication

Under mu, create the generation context and invoke the factory. New not-ready
callers may add reservations, but they cannot copy fields while initialization
is writing them. Store the complete entry and control data before setting ready
with an atomic Or, preserving the counts already added by other callers.

Ordinary factory errors are completed results and remain cached until Reset.
Cancel the failed generation and close any partial dialer. Permanent Close wins
over factory success through the lifetime context and retirement checks.

The current code leaves an ErrCloseRequested factory result uncached, allowing
the pool's setup Disconnect to remain reconnectable. Keep this explicit exception
until its error-policy boundary is agreed; do not silently cache it or conflate
it with permanent retirement.

A successful generation starts its Wait observer. That observer must not mutate
the shared entry. It ends the captured generation and signals completion after
the pool's Wait wrapper has queued Offline. A coalesced activation notification
lets the cleanup loop watch the currently published generation. It must not be
a blocking send while the factory holds mu.

## Decrements and idle timing

Queued releases stay in the packed state until the consumer applies them. When
a decrement containing a reference portion makes references zero for an active
generation, the loop starts a fresh full idle interval. Reader-only completion
does not restart the idle interval.

Activation must also handle the case where references became zero before the
loop processed activation. If duration is nonpositive, idle timeout is disabled;
death, Reset, and Close still work.

An acquisition need not stop the timer. Its atomic Add makes the idle CAS fail.
If the timer event is processed while references remain, cleanup does nothing;
the later zero-reference decrement supplies the fresh interval. Reuse one timer
and stop or disable its watch when its generation is invalidated.

This proof assumes the modern channel-timer behavior used by the module's Go
1.26 requirement. After Reset or Stop returns, later receives cannot deliver an
old timer setting. A timeout already received is instead ordered by the single
consumer. Do not claim this proof for a runtime explicitly opting into legacy
timer-channel behavior. See [Timer.Reset](https://pkg.go.dev/time#Timer.Reset)
and [Timer.Stop](https://pkg.go.dev/time#Timer.Stop).

## Reset death and permanent Close

In the current candidate, for Reset or detected death, the cleanup loop takes mu,
clears ready, drains reader decrements, cancels the old generation, closes its dialer, and observes
Wait completion. Then it clears the old fields, disables old watches, and
unlocks. A waiting request may initialize the next generation immediately.

Here Reset is processed against the result current when it holds mu. It returns
false if nothing is published or the lazy dialer is permanently closed. Waiting
for an initializing factory is the tradeoff described above; it is not required
by the Reset goal. Concurrent requests may reinitialize after Reset's clearing
operation, so completion does not promise that the lazy dialer remains empty.

Idle cleanup follows the same teardown only after the zero-count CAS succeeds.
Permanent Close cancels the immutable lifetime context immediately. The worker
then excludes readers, tears down the generation, leaves a terminal result, and
continues consuming outstanding releases until it can retire.

Old connection releases belong to the aggregate count even after a replacement
starts. They can conservatively delay its idle timeout. The timer uses the
current generation's watch, so an old release cannot independently close the
replacement or cancel its context.

## Race catalogue

Each pair gives both orderings at the relevant synchronization point. A physical
network failure or wall-clock event is not itself an atomic publication or idle
claim. Cases marked as proof obligations are not yet closed by this draft.

### 1 Two callers see not ready

- A acquires mu first: A initializes; B rechecks ready after acquiring mu and
  uses A's published result. Safe because B does not write it again.
- B acquires mu first: the same sequence with the roles reversed. Safe because
  atomic Add does not elect an initializer; mutex ownership and the recheck do.

### 2 Acquisition races with ready publication

- Add precedes publication: the caller sees not ready, queues its reader release,
  and waits on mu. Safe because initialization excludes its eventual copy.
- Publication precedes Add: the caller sees ready and copies the completed entry.
  Safe because the atomic publication orders preceding field writes.

### 3 Acquisition races with invalidation

- Add observes ready before it is cleared: the writer must wait for that snapshot
  reader to finish. Its local copy remains the old generation. Safe replacement
  does not rewrite the captured result.
- Ready is cleared before Add: the caller does not copy fields; it queues its
  reader release and waits on mu. Safe because it can only copy after teardown.

### 4 Reader completion races with reader draining

- Decrement is consumed first: the reader is already absent when the writer
  checks the count. Safe because the copy preceded the send.
- Writer checks first: the count remains positive until the writer consumes the
  queued decrement. Safe because it does not overwrite fields prematurely.

### 5 Not ready reader release races with a waiting writer

- Release is consumed before the caller waits on mu: that caller is not counted
  as a reader anymore. Safe; only its connection reservation remains.
- Caller is parked on mu before consumption: its reader message is already
  available to the cleanup drain. Safe for finite arrivals; it is not waiting
  for mu to send the decrement. See sustained-arrival proof obligation below.

### 6 Acquisition races with an idle claim

- Add wins: the word has nonzero counts, so CAS(readyBit, 0) fails. Safe because
  the acquired generation is not closed for idleness.
- CAS wins: Add sees not ready and waits behind teardown. Safe because it never
  reads the claimed generation and the next initialization uses a new context.

### 7 Last connection release races with timeout handling

- Timeout handler wins: the unconsumed release is still counted, so the CAS
  fails. Consuming that release later resets a full interval. Safe: no premature
  idle close, even if Close occurred before the timer handler ran.
- Decrement handler wins: it makes references zero and resets the timer before
  another receive. Safe under modern timer semantics; an expired old interval
  is not reused to immediately close the newly idle generation.

### 8 A short connection begins and ends while an old timer expires

- Timeout processing precedes release processing: Add has already reserved the
  connection, and its decrement is still queued, so the idle CAS fails. Safe.
- Release processing precedes timeout processing: the consumer restarts the
  interval. Safe: the connection's recent activity is not erased by a callback
  that started or became runnable earlier.

### 9 Reader completion races with an idle timer

- Reader completion is consumed first: the connection reservation still remains,
  so the idle CAS cannot succeed. Safe; copying is not the end of connection use.
- Timeout is processed first: both the reader and reference remain counted, so
  the CAS fails. Safe; a reader-only decrement never falsely starts inactivity.

### 10 A dial fails while idle timeout is pending

- Timeout runs first: the failed dial's reference has not been released by the
  consumer, and cleanup cannot claim zero. Safe.
- Failure release is consumed first: references reaching zero restart the idle
  interval. Safe; the failed attempt counts as recent use of the active tunnel.

### 11 Release races with generation activation

- Release is consumed first: activation must notice zero references and arm an
  interval itself. Safe only if this branch is implemented, not assumed.
- Activation is processed first: the zero-reference release arms or resets its
  timer normally. Safe; activation and decrement timing share one consumer.

### 12 Reset races with acquisition

- Acquisition sees ready first: Reset drains its snapshot reader, then cancels
  that old context. The request either fails dialing or its returned wrapper
  closes. Safe: Reset is allowed to disconnect a concurrent old-generation user.
- Reset clears ready first: acquisition waits on mu and initializes or uses the
  replacement after teardown. Safe: it does not read a partially cleared entry.

### 13 Reset races with an initializing factory

- Reset gets mu before any initializer: it clears any existing result or reports
  nothing to reset, then a later request initializes. Safe; Reset need not keep
  the lazy dialer empty against future requests.
- Factory gets mu first: this candidate makes Reset wait, then invalidate the
  result. Reference and publication safety depend on setup terminating or
  lifetime Close cancelling it; responsiveness remains a tradeoff. A design
  that cancels setup instead must also prevent its late result from surviving
  Reset. Both approaches must satisfy that goal before either is chosen.

### 14 Reset races with timeout or detected death

- Reset claims the current result first: teardown finishes once; old watches are
  disabled. Safe: an old event must not target a replacement generation.
- Timeout or death claims first: Reset sees no result, or a replacement that
  was initialized before it got mu. Reset may clear that replacement. Safe:
  each operation is ordered at its locked check, not its invocation timestamp.

### 15 Two Reset requests overlap

- First request finishes with no intervening initialization: second reports
  nothing to reset. Safe: the old generation is not torn down twice.
- A new request initializes between them: second resets that new generation.
  Safe: two full Reset operations may disconnect two different current results.

### 16 Tunnel death races with acquisition

- Acquisition copies before readiness is invalidated: it can acquire a result
  whose transport has just died. Safe as a snapshot, but its dial must tolerate
  failure and generation cancellation. Ready is not a health guarantee.
- Invalidation precedes acquisition: the caller waits for cleanup and obtains
  a replacement. Safe: it does not copy fields being cleared.

There is necessarily a detection window between physical death, Wait returning,
and the worker clearing ready. Do not label that window a memory race or promise
that a live transport cannot die immediately after successful acquisition.

### 17 Old Wait completion races with replacement initialization

- Wait completes first: teardown observes its completion, clears the old entry,
  and permits the new factory. Safe; the old Offline update precedes Connecting.
- Initialization tries first: it waits on mu until old Wait completion has been
  observed. Safe; the old observer only holds the old cancel function and does
  not later mutate the replacement's fields.

### 18 Old generation release races with new generation activity

- Old release is consumed first: the aggregate count drops before the new Add
  and activation. Safe; the new reservation protects its eventual idle claim.
- New Add happens first: consuming the old release subtracts only its own unit,
  leaving the new reference counted. Safe because replacement never zeroed the
  aggregate count. An old final release can extend, not shorten, idle timing.

### 19 Caller cancellation races with dial completion

- Cancellation is observed before success is handed off: return the error, close
  any partial connection, and queue exactly one release. Safe: no leaked socket
  or reserved reference. Audit the post-dial check during implementation.
- Success is handed off first: caller cancellation no longer owns the returned
  connection; its owner closes it later. Safe because request contexts are not
  connection reference counters. Cancellation concurrent with the handoff may
  resolve either way, but neither outcome may leak a reference.

### 20 Generation cancellation races with connection handoff

- Cancellation is observed first: the dial fails or its partial result is closed
  and released. Safe; a cancelled generation cannot leave an unowned connection.
- Wrapper is handed off first: its registered generation callback closes it.
  Safe; an already-cancelled context also runs an AfterFunc registered late.

### 21 Explicit Close races with automatic connection close

- Explicit Close wins: stop the cancellation callback if possible, close the
  connection, and release once. Safe even if stopping the callback loses.
- Callback wins: explicit Close may arrive later, but release still happens once.
  Safe for accounting. Underlying Close idempotency must also be established;
  a once-only decrement alone does not prove arbitrary Close implementations safe.

### 22 Permanent Close races with a factory result

- Lifetime cancellation is observed before publication: dispose of any partial
  result and fail; the retirement handler prevents later initialization. Safe.
- Publication occurs first, including a cancellation just after the factory's
  cause check: its child context is cancelled and retirement tears it down.
  Safe only if callers check lifetime cancellation and wrappers observe the
  captured generation. Publication must not resurrect the lifetime context.

### 23 Permanent Close races with Reset

- Reset runs first: it may complete, but subsequent lifetime cancellation retires
  the lazy dialer and any replacement. Safe: Reset cannot undo Close.
- Close cancels first: Reset returns false or its queued request is answered as
  false. Safe: retirement must also wake Reset senders when the worker exits.

### 24 Permanent Close races with connection release

- Release is consumed first: fewer references remain when retirement starts.
  Safe; teardown still excludes readers before replacing fields.
- Retirement starts first: generation cancellation closes wrappers, and the
  worker remains available to consume their releases. Safe only if teardown
  does not block the consumer behind those release producers; see below.

### 25 A delayed acquisition races with worker retirement

- Add happens before the worker's final zero-count observation: retirement must
  consume its reader/reference releases before exiting. Safe.
- Caller passed the lifetime check but Add occurs after that observation: it
  must see the terminal result or be rejected under mu, and its decrement send
  must not block forever after done is closed. Safe for permanent retirement,
  where remaining packed counts no longer govern a reusable entry. Do not
  claim that the terminal counter necessarily remains numerically zero.

### 26 A stale event races with a new generation

- Old event is handled before replacement: it targets its captured old context.
  Safe; normal teardown disables that generation's watches.
- Replacement is published first: reject the old event by generation identity,
  or ensure the old watch was disabled before replacement became possible.
  Safe only with that check; a ready-only CAS alone cannot distinguish generations.

### 27 Cached failure races with error Reset and new callers

- Caller copies the error first: it returns that error and queues both decrements.
  Safe; Reset waits for its copy, not for the caller to finish reporting it.
- Reset clears readiness first: caller waits and may initialize again. Safe;
  cached failure is not mutated under an unprotected reader.

### 28 Backpressure and sustained arrivals

- Consumer remains runnable and arrivals become finite: blocked senders can
  enqueue releases; the drain reaches zero and writers complete. Safe, subject
  to normal scheduling and eventual Close/Wait completion.
- Consumer waits on code needing those same senders, or new not-ready Add calls
  outpace the drain indefinitely: the buffer does not prove progress. This is
  a liveness proof obligation, not an acceptable outcome to dismiss as a slow
  path. Address it before declaring the design complete.

### 29 Caller cancellation races with shared initialization

- Caller cancellation precedes acquisition: reject it without adding counts or
  starting a factory. Safe; it owns no reference to release.
- Cancellation occurs after reservation or while waiting for the factory: shared
  initialization may finish because it uses the generation context, not that
  caller's context. Reject the cancelled caller before handing it a connection
  and release its reference. Safe; other requests may use the published result.
  Caller cancellation alone must not become a cached factory failure.

### 30 Cleanup races with a slow path copy under mu

- Slow caller holds mu first: it copies the entry into its local value before
  unlocking. Cleanup can then replace the fields without affecting that copy.
  Safe even if the caller's queued reader decrement was already consumed.
- Cleanup holds mu first: the slow caller waits through draining and teardown,
  then sees or initializes the new result. Safe; the reader count protects
  fast copies, and mutex ownership protects slow copies.

### 31 Factory failure races with cancellation

- Ordinary failure is established first with the generation still valid: publish
  a cached error and balance its request reservations. Safe; later callers see
  the same completed failure until Reset rather than invoking factories in parallel.
- Cancellation is observed before publication: dispose of any partial result
  and use the cancellation outcome. Safe only with the explicit distinction
  between permanent lifetime retirement and the existing uncached setup
  Disconnect exception. Neither outcome may publish a usable partial dialer.

### 32 Idle timeout races with natural tunnel completion

- Idle cleanup claims first: it cancels and closes the captured generation and
  observes Wait completion. Safe; a subsequent completion notification cannot
  tear down a replacement.
- Wait completion is observed first: the dead-generation cleanup owns teardown
  and disables the old timer watch. Safe; no second teardown is needed. The first
  context cancellation cause can determine the displayed reason in this race;
  accounting and generation ownership cannot depend on which reason wins.

## Proof obligations found in the current implementation

### Teardown must not stop the only decrement consumer

The current clear helper drains readers, then calls dialer.Close and waits for
finished without receiving further decrement messages. Cancellation callbacks
and new requests can fill the 256-slot buffer during that period. If Close or
Wait depends on a blocked sender, this is a wait cycle. Simply making the buffer
larger is not a proof.

During implementation, trace each supported Close and Wait path. Either establish
that Close cannot depend on these producers, or initiate teardown asynchronously
and let the same cleanup worker keep consuming decrements until the old Wait
completion arrives. A wait-for-finished phase should consume queued releases.
Do not add a second decrement consumer or a goroutine per ordinary release.

### Reader drain is conservative not writer priority

With one Add and all decrements queued, new callers observing not ready still
briefly contribute to the reader count. Safety requires draining before field
replacement, but the count does not isolate only readers admitted before ready
was cleared. Finite concurrency drains correctly. A bounded writer-latency claim
under sustained traffic needs an additional proof or a revised admission scheme.
Do not implement new flags or counter fields without agreeing that tradeoff.

### Reset and UI failure ordering

The menu-state loop must never synchronously wait for a full Lazy.Reset: Wait
queues Offline into that same loop. A reset of an already displayed cached
failure can queue Offline before invalidation, keeping replacement Connecting
behind it. A reset while setup is still running can finish by clearing a newly
published failure after Failed was queued. The current pre-reset UI message does
not by itself prove a final Offline state for that case. Resolve this ordering
without letting an old reset completion overwrite a replacement's newer status.
Do not add a generic UI hook to Lazy as a shortcut.

### Completion and double close audit

The current dial helper delegates directly to context-aware dialers, and its
fallback select can have cancellation and success ready together. Inspect the
handoff and disposal path, including a non-nil connection returned with an error.
Also verify repeated and concurrent underlying Close behavior for SSH,
OpenConnect, forwarding, and returned connection wrappers. Once-only ref release
is already required; it is not the whole resource-lifetime proof.

## Implementation sequence after design review

1. Review the goals, decide cancellation responsiveness during initialization,
   and close the proof obligations above. Choose the design from those needs;
   record any additional synchronization and the concrete problem it solves.
2. Introduce the entry snapshot and reduce acquire to entry plus release. Preserve
   one acquisition Add and all-channel decrements; remove only newly orphaned
   fields, returns, helpers, and comments. Verify every field access against the
   publication/draining invariants.
3. Make initialization and cleanup linear: acquire mu, recheck, initialize or
   exclude/drain/teardown, publish or clear, unlock. Preserve aggregate counts and
   the immutable lifetime context. Implement the Reset behavior selected during
   design review rather than assuming it must wait for setup.
4. Make one decrement application path serve the main loop and teardown waits.
   Keep the 256-slot channel and reusable timer. Verify race cases 6 through 11,
   18, 24, and 28 against the resulting source.
5. Wire the connection wrapper and pool status ownership. Resolve handoff,
   double-close, and Reset notification sequencing. Verify race cases 13 and
   17 through 25 against the actual callback and cancellation paths.
6. Perform formatting, git diff --check, the full Pacman Nix-shell build, and
   focused go vet. Do not add or run tests without the user's explicit request.
   Static verification and a build do not prove runtime race freedom; report the
   remaining assumptions and any approved live checks separately.

## Source checkpoints

- `pkg/dialer/lazy.go`: packed state, acquisition, contexts, queued decrements,
  initialization, timer, Reset, retirement, and reader drain.
- `internal/app/pool.go`: factory/UI ownership, setup cancellation, Wait ordering,
  and Reset notification handling.
- `internal/app/app.go`: stable lazy dialers captured by rules and proxy chains.
- `pkg/dialer/common.go` and `pkg/dialer/ssh.go`: setup cancellation and successful
  connection handoff.
- `pkg/dialer/dailer.go`, `pkg/dialer/oc/oc.go`, and
  `pkg/openconnect/client.go`: native and forwarding Close/Wait behavior.
- `pkg/httpproxy/server.go`: outbound keep-alive disabled; upstream connections
  still release through their real Close path, not response context cancellation.

Review gate: every implemented transition must map to an invariant and the
corresponding race pair. A missing outcome or an unproved dependency belongs in
this plan before it becomes another patch to the implementation.
