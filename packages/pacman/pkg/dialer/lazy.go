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
	new     func() (proxy.ContextDialer, error)
	timeout time.Duration

	mu      sync.RWMutex
	cond    sync.Cond
	initing atomic.Bool

	xd    proxy.ContextDialer
	err   error
	state ConnectionState

	ctx       context.Context
	cancel    context.CancelCauseFunc
	timer     *time.Timer
	timerRace atomic.Bool
}

func NewLazy(new func() (proxy.ContextDialer, error), timeout time.Duration) *Lazy {
	l := Lazy{
		new:     new,
		timeout: timeout,
	}
	l.cond.L = l.mu.RLocker()
	return &l
}

func (d *Lazy) Dial(network, addr string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr)
}

func (d *Lazy) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var waited bool

	d.mu.RLock()
	for {
		state := d.state
		switch state {
		case Online:
			if d.timer != nil && !d.timer.Stop() {
				d.timerRace.Store(true)
			}
			ctx = contextutil.Merge(ctx, d.ctx)
			conn, err := d.xd.DialContext(ctx, network, addr)
			context.AfterFunc(ctx, func() {
				d.timer.Reset(d.timeout)
				d.mu.RUnlock()
			})
			return conn, err
		case Offline, Failed:
			if waited {
				defer d.mu.RUnlock()
				return nil, d.err
			}
			if d.initing.CompareAndSwap(false, true) {
				go d.init()
			}
		case Connecting:
		default:
			panic("invalid dialer state")
		}

		d.cond.Wait()
		waited = true
	}
}

func (d *Lazy) State() ConnectionState {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.state
}

func (d *Lazy) Subscribe(yield func(ConnectionState, error) bool) {
	for {
		d.mu.RLock()
		d.cond.Wait()
		state, err := d.state, d.err
		d.mu.RUnlock()
		if !yield(state, err) {
			return
		}
	}
}

func (d *Lazy) Close() error {
	// signal to stop any ongoing operations
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.cancel != nil {
		d.cancel(ErrCloseRequested)
	}
	d.cond.Broadcast()
	return nil
}

func (d *Lazy) init() {
	d.mu.Lock()
	d.xd, d.state, d.err = nil, Connecting, nil
	d.initing.CompareAndSwap(true, false)
	d.cond.Broadcast()
	d.mu.Unlock()

	xd, err := d.new()
	if err != nil {
		d.mu.Lock()
		d.xd, d.state, d.err = nil, Failed, err
		d.cond.Broadcast()
		d.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	timer := time.NewTimer(d.timeout)

	if w, ok := xd.(waiter); ok {
		go func() {
			w.Wait()
			cancel(ErrUnderlyingClosed)
		}()
	}

	d.mu.Lock()
	d.xd, d.state, d.err = xd, Online, err
	d.ctx, d.cancel = ctx, cancel
	d.timer = timer
	d.timerRace.Store(false)
	d.cond.Broadcast()
	d.mu.Unlock()

	go func() {
		var tc <-chan time.Time
		if d.timeout > 0 {
			tc = d.timer.C
		}

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

		cause := context.Cause(ctx)
		switch cause {
		case ErrCloseRequested, ErrIdleTimeout:
			if c, ok := d.xd.(io.Closer); ok {
				go c.Close()
			}
		}

		d.xd, d.state, d.err = nil, Offline, cause
		d.ctx, d.cancel = nil, nil
		d.timer = nil
		d.timerRace.Store(false)
		d.cond.Broadcast()
		d.mu.Unlock()
	}()

	return
}
