package dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gilliginsisland/pacman/pkg/syncutil"
	"golang.org/x/net/proxy"
)

var (
	ErrCloseRequested = errors.New("close requested")
	ErrResetRequested = errors.New("reset requested")
)

// Acquisition observes readiness and takes a reference in one atomic operation.
// References cover active dials/connections and transient not-ready acquisitions.
// Ready releases remain counted until the monitor consumes their notifications.
const lazyReadyBit uint64 = 1 << 63

// ResetAction preserves the caller's intent if the state changes before delivery.
type ResetAction uint8

const (
	Cancel ResetAction = iota + 1
	Disconnect
	Reset
	Connect
)

type LazyOptions struct {
	IdleTimeout time.Duration
	// Updates receives state snapshots without blocking tunnel lifecycle work.
	// If the receiver is busy, the latest snapshot replaces any pending update.
	Updates chan<- StateUpdate
}

// Dialer owns its in-progress dials and returned connections: Close must stop
// both and make Wait return. Close may be called again after Wait has returned.
type Dialer interface {
	proxy.Dialer
	proxy.ContextDialer
	io.Closer
	Wait() error
}

type Lazy struct {
	// Configuration, lifetime, and channel references are immutable after construction.
	new          func(context.Context) (Dialer, error)
	idleTimeout  time.Duration
	stateUpdates chan<- StateUpdate

	// Shared state: use atomic operations on state. Only monitor writes value
	// and err; callers may read them while holding a ready reference.
	state atomic.Uint64
	value Dialer
	err   error

	requests *syncutil.Switchable[chan struct{}]
	releases chan uint64
	resets   *syncutil.Switchable[chan ResetAction]

	// Monitor-owned state: only monitor and its lifecycle helpers mutate these fields.
	// Workers only send on wait; that channel is never replaced.
	factory *syncutil.Deferred[Dialer, error]
	update  StateUpdate
	updates chan<- StateUpdate
	// rundown counts not-ready acquisitions caught between Add and release,
	// not callers parked on requests (which have already released).
	rundown uint64
	wait    *syncutil.Switchable[chan error]
	timer   *time.Timer
	timeout *syncutil.Switchable[<-chan time.Time]

	ctx    context.Context
	cancel context.CancelCauseFunc
}

// NewLazy caches the factory result. On failure, the factory must clean up its
// resources and return a nil dialer. Its context covers setup only: Deferred
// cancels it after completion, so a successful dialer must own its own lifetime.
func NewLazy(new func(context.Context) (Dialer, error), options LazyOptions) *Lazy {
	ctx, cancel := context.WithCancelCause(context.Background())
	l := Lazy{
		new:          new,
		idleTimeout:  options.IdleTimeout,
		stateUpdates: options.Updates,
		ctx:          ctx,
		cancel:       cancel,
		requests:     syncutil.NewSwitchable(make(chan struct{})),
		releases:     make(chan uint64, 256),
		resets:       syncutil.NewSwitchable(make(chan ResetAction, 1)),
		wait:         syncutil.NewSwitchable(make(chan error, 1)),
	}
	l.requests.Enable()
	l.resets.Enable()
	l.wait.Enable()
	var timeout <-chan time.Time
	if options.IdleTimeout != 0 {
		l.timer = time.NewTimer(options.IdleTimeout)
		timeout = l.timer.C
		l.timer.Stop()
	}
	l.timeout = syncutil.NewSwitchable(timeout)
	l.timeout.Enable()
	go l.monitor()
	return &l
}

func (d *Lazy) Dial(network, addr string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr)
}

// DialContext holds a tunnel reference until the returned connection is closed.
// Callers must Close even after EOF; request context cancellation is not a release.
func (d *Lazy) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	value, release, err := d.acquire(ctx)
	if err != nil {
		return nil, err
	}

	conn, err := value.DialContext(ctx, network, addr)
	if err != nil {
		release()
		return nil, err
	}

	return &trackedConn{
		Conn:    conn,
		onClose: release,
	}, nil
}

func (d *Lazy) acquire(ctx context.Context) (Dialer, func(), error) {
	for {
		if d.state.Add(1)&lazyReadyBit != 0 {
			value, err := d.value, d.err
			if err != nil {
				d.release(lazyReadyBit)
				return nil, nil, err
			}
			return value, sync.OnceFunc(func() { d.release(lazyReadyBit) }), nil
		}
		// Release before waiting. A retry acquires a ready reference before
		// reading the shared result.
		d.release(0)
		var err error
		select {
		// Only the monitor's offers are switchable. Callers always wait on the
		// stored channel so enabling offers can wake an already parked caller.
		case <-d.requests.Value():
		case <-ctx.Done():
			err = ctx.Err()
		case <-d.ctx.Done():
			err = context.Cause(d.ctx)
		}
		if err != nil {
			return nil, nil, err
		}
	}
}

// Ready releases stay counted until the monitor can restart the idle timer.
// Not-ready releases only notify when they acknowledge a publication snapshot.
// The payload is the observed ready bit: zero has already decremented state,
// while a ready-tagged message still owes the monitor that decrement.
func (d *Lazy) release(ready uint64) {
	if ready == 0 && d.state.Add(^uint64(0))&lazyReadyBit == 0 {
		return
	}
	select {
	case d.releases <- ready:
	case <-d.ctx.Done():
	}
}

// Close permanently cancels factories, in-progress dials, and connections.
func (d *Lazy) Close() {
	d.cancel(ErrCloseRequested)
}

// Done is closed when permanent Close is requested, not on idle retirement.
func (d *Lazy) Done() <-chan struct{} {
	return d.ctx.Done()
}

// Reset queues an action, which is ignored if it no longer applies. Cancel only
// cancels setup, Disconnect only closes a tunnel, Reset only clears an error, and
// Connect only starts an empty dialer. Different actions are not coalesced.
func (d *Lazy) Reset(action ResetAction) {
	select {
	case d.resets.Value() <- action:
	case <-d.ctx.Done():
	}
}

func (d *Lazy) monitor() {
	if d.timer != nil {
		defer d.timer.Stop()
	}
	for {
		select {
		case d.updates <- d.update:
			d.updates = nil
		case d.requests.Current() <- struct{}{}:
			// A rendezvous either starts shared setup or lets a slow caller retry.
			if d.state.Load()&lazyReadyBit != 0 {
				break
			}
			d.requests.Disable()
			d.factory = syncutil.NewDeferred(d.ctx, d.new)
			d.notify(Connecting, nil)
		case <-d.factory.Done():
			value, err := d.factory.Values()
			d.factory = nil
			if err == nil {
				go func() {
					// The previous wait result is consumed before another factory
					// starts, so the one-slot buffer also permits sending after exit.
					d.wait.Value() <- value.Wait()
				}()
			}
			d.value, d.err = value, err
			// Old not-ready observers must acknowledge publication before
			// any lifecycle event can clear readiness again.
			// Delaying death handling may expose an already-dead dialer until
			// these observers release, preventing overlap of not-ready cohorts.
			d.rundown = d.state.Or(lazyReadyBit) &^ lazyReadyBit
			if d.rundown != 0 {
				d.resets.Disable()
				d.wait.Disable()
				d.timeout.Disable()
			}
			d.requests.Enable()
			d.resetIdleTimer()
			if err != nil {
				d.notify(Failed, err)
			} else {
				// Online means shared setup succeeded; stateless dialers may
				// not contact their upstream until DialContext is called.
				d.notify(Online, nil)
			}
		case err := <-d.wait.Current():
			if !d.retire() {
				return
			}
			d.value, d.err = nil, nil
			d.requests.Enable()
			d.notify(Offline, err)
		case ready := <-d.releases:
			if ready != 0 {
				d.state.Add(^uint64(0))
			} else if d.rundown != 0 {
				d.rundown--
				if d.rundown == 0 {
					d.resets.Enable()
					d.wait.Enable()
					d.timeout.Enable()
				}
			}
			d.resetIdleTimer()
		case <-d.timeout.Current():
			// Queued ready releases remain counted until processed above. A
			// stale timeout therefore cannot win before the last release resets it.
			if !d.state.CompareAndSwap(lazyReadyBit, 0) {
				continue
			}
			d.requests.Disable()
			go d.value.Close()
		case action := <-d.resets.Current():
			switch action {
			case Cancel:
				if d.factory != nil {
					select {
					case <-d.factory.Done():
					default:
						d.factory.Cancel(ErrResetRequested)
					}
				}
			case Disconnect:
				if d.value != nil {
					if !d.retire() {
						return
					}
				}
			case Reset:
				if d.err != nil {
					if !d.retire() {
						return
					}
					d.err = nil
					d.requests.Enable()
					d.notify(Offline, nil)
				}
			case Connect:
				if d.factory == nil && d.value == nil && d.err == nil {
					d.requests.Disable()
					d.factory = syncutil.NewDeferred(d.ctx, d.new)
					d.notify(Connecting, nil)
				}
			}
		case <-d.ctx.Done():
			if factory := d.factory; factory != nil {
				// The monitor is exiting, so it can no longer take ownership of this result.
				go func() {
					<-factory.Done()
					if value, err := factory.Values(); err == nil {
						value.Close()
					}
				}()
			}
			if d.state.And(^lazyReadyBit)&lazyReadyBit != 0 && d.value != nil {
				d.value.Close()
			}
			return
		}
	}
}

func (d *Lazy) notify(state ConnectionState, err error) {
	d.update = StateUpdate{State: state, Err: err}
	select {
	case d.stateUpdates <- d.update:
		d.updates = nil
	default:
		// Retry from the monitor select so an idle final state is not lost.
		d.updates = d.stateUpdates
	}
}

func (d *Lazy) resetIdleTimer() {
	// Errors stay cached until Reset; idle expiry only retires live dialers.
	if d.timer != nil && d.state.Load() == lazyReadyBit && d.err == nil {
		d.timer.Reset(d.idleTimeout)
	}
}

// retire blocks new field readers and drains existing ready references.
// If Wait has not returned, monitor continues waiting for it before allowing setup.
// False means permanent cancellation interrupted the drain, so monitor can exit.
func (d *Lazy) retire() bool {
	d.requests.Disable()
	if state := d.state.And(^lazyReadyBit); state&lazyReadyBit != 0 {
		if d.value != nil {
			go d.value.Close()
		}
		// Publication rundown has already consumed every zero-tagged message.
		// New not-ready callers release directly, so these are all ready releases;
		// Discard consumes their notifications and we subtract the batch once.
		n := syncutil.Discard(d.ctx, d.releases, state&^lazyReadyBit)
		d.state.Add(-n)
		if n != state&^lazyReadyBit {
			return false
		}
	}
	if d.timer != nil {
		d.timer.Stop()
	}
	return true
}
