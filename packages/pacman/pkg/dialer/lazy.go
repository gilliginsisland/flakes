package dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/contextutil"
	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

var (
	ErrUnderlyingClosed = errors.New("underlying dialer closed")
	ErrCloseRequested   = errors.New("close requested")
	ErrIdleTimeout      = errors.New("idle timeout reached")
)

type waiter interface {
	Wait() error
}

type ConnectionState int

const (
	Offline ConnectionState = iota
	Connecting
	Failed
	Online
)

type StateSignal struct {
	State ConnectionState
	Err   error
}

type Lazy struct {
	Timeout time.Duration
	New     func() (proxy.ContextDialer, error)

	mu      sync.RWMutex // protects all state
	initing atomic.Bool  // Tracks if initialization is in progress

	xd     proxy.ContextDialer
	err    error
	state  ConnectionState
	signal syncutil.Signal[StateSignal]

	ctx       context.Context
	cancel    context.CancelCauseFunc
	timer     *time.Timer
	timerRace atomic.Bool
}

func (d *Lazy) Dial(network, addr string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr)
}

func (d *Lazy) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var ch <-chan StateSignal
	for {
		d.mu.RLock()
		state := d.state
		switch state {
		case Online:
			if d.timer != nil && !d.timer.Stop() {
				d.timerRace.Store(true)
			}
			ctx = contextutil.Merge(ctx, d.ctx)
			conn, err := d.xd.DialContext(ctx, network, addr)
			context.AfterFunc(ctx, func() {
				d.timer.Reset(d.Timeout)
				d.mu.RUnlock()
			})
			return conn, err
		case Failed:
			if ch != nil {
				defer d.mu.RUnlock()
				return nil, d.err
			}
			fallthrough
		case Offline:
			if d.initing.CompareAndSwap(false, true) {
				go d.init()
			}
		case Connecting:
		default:
			panic("invalid dialer state")
		}

		if ch == nil {
			var cancel func()
			ch, cancel = d.signal.Subscribe()
			defer cancel()
		}
		d.mu.RUnlock()
		<-ch
	}
}

func (d *Lazy) State() ConnectionState {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.state
}

func (d *Lazy) Subscribe() (<-chan StateSignal, func()) {
	return d.signal.Subscribe()
}

func (d *Lazy) Close() error {
	// signal to stop any ongoing operations
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.cancel != nil {
		d.cancel(ErrCloseRequested)
	}
	return nil
}

func (d *Lazy) init() {
	d.mu.Lock()
	d.state, d.err = Connecting, nil
	d.signal.Publish(StateSignal{State: d.state, Err: d.err})
	d.initing.CompareAndSwap(true, false)
	d.mu.Unlock()

	xd, err := d.New()
	if err != nil {
		d.xd, d.state, d.err = nil, Failed, err
		d.signal.Publish(StateSignal{State: d.state, Err: d.err})
		return
	}

	d.xd, d.state, d.err = xd, Online, err
	d.signal.Publish(StateSignal{State: d.state, Err: d.err})

	ctx, cancel := context.WithCancelCause(context.Background())
	timer := time.NewTimer(d.Timeout)

	var tc <-chan time.Time
	if d.Timeout > 0 {
		tc = timer.C
	}

	d.ctx, d.cancel = ctx, cancel
	d.timer = timer
	d.timerRace.Store(false)

	if w, ok := d.xd.(waiter); ok {
		go func() {
			w.Wait()
			cancel(ErrUnderlyingClosed)
		}()
	}

	go func() {
		for {
			select {
			case <-tc:
				d.mu.Lock()
				if d.timerRace.Load() {
					// if timerRace is true a reader reset the timer
					d.mu.Unlock()
					continue
				}
				cancel(ErrIdleTimeout)
			case <-ctx.Done():
				d.mu.Lock()
			}
			break
		}
		defer d.mu.Unlock()

		cause := context.Cause(ctx)
		switch cause {
		case ErrCloseRequested, ErrIdleTimeout:
			if c, ok := d.xd.(io.Closer); ok {
				go c.Close()
			}
		}

		d.ctx, d.cancel = nil, nil
		d.timer = nil
		d.timerRace.Store(false)

		d.xd, d.state, d.err = nil, Offline, cause
		d.signal.Publish(StateSignal{State: d.state, Err: d.err})
	}()

	return
}
