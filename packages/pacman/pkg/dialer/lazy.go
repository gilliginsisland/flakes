package dialer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/contextutil"
)

var (
	ErrCloseRequested = errors.New("close requested")
	ErrIdleTimeout    = errors.New("idle timeout reached")
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
	new     func(ctx context.Context) (proxy.Dialer, error)
	timeout time.Duration

	mu      sync.RWMutex
	cond    sync.Cond
	initing atomic.Bool

	xd    proxy.Dialer
	err   error
	state ConnectionState

	ctx       context.Context
	cancel    context.CancelCauseFunc
	timer     *time.Timer
	timerRace atomic.Bool
}

func NewLazy(new func(ctx context.Context) (proxy.Dialer, error), timeout time.Duration) *Lazy {
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
			conn, err := dialContext(ctx, d.xd, network, addr)
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
	} else {
		d.cond.Broadcast()
	}
	return nil
}

func (d *Lazy) init() {
	ctx, cancel := context.WithCancelCause(context.Background())
	timer := time.NewTimer(d.timeout)

	d.mu.Lock()
	d.xd, d.state, d.err = nil, Connecting, nil
	d.ctx, d.cancel, d.timer = ctx, cancel, timer
	d.timerRace.Store(false)
	d.initing.CompareAndSwap(true, false)
	d.cond.Broadcast()
	d.mu.Unlock()

	xd, err := d.new(ctx)
	if err != nil {
		d.mu.Lock()
		d.xd, d.state, d.err = nil, Failed, err
		d.ctx, d.cancel, d.timer = nil, nil, nil
		d.cond.Broadcast()
		d.mu.Unlock()
		return
	}

	if w, ok := xd.(waiter); ok {
		go func() {
			cancel(fmt.Errorf("underlying dialer closed: %w", w.Wait()))
		}()
	}

	d.mu.Lock()
	d.xd, d.state, d.err = xd, Online, err
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
		d.ctx, d.cancel, d.timer = nil, nil, nil
		d.timerRace.Store(false)
		d.cond.Broadcast()
		d.mu.Unlock()
	}()

	return
}
