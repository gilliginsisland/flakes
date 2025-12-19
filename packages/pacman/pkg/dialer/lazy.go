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
	"github.com/gilliginsisland/pacman/pkg/syncutil"
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
	new      func(ctx context.Context) (proxy.Dialer, error)
	duration time.Duration

	mu      sync.RWMutex
	cond    sync.Cond
	initing atomic.Bool

	xd    proxy.Dialer
	err   error
	state ConnectionState

	ctx     context.Context
	cancel  context.CancelCauseFunc
	timeout *syncutil.Timeout
}

func NewLazy(new func(ctx context.Context) (proxy.Dialer, error), duration time.Duration) *Lazy {
	l := Lazy{
		new:      new,
		duration: duration,
	}
	l.cond.L = l.mu.RLocker()
	return &l
}

func (d *Lazy) Dial(network, addr string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr)
}

func (d *Lazy) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	d.mu.RLock()
	for {
		state := d.state
		switch state {
		case Offline:
			if d.initing.CompareAndSwap(false, true) {
				go d.init()
			}
		case Online:
			if d.duration > 0 {
				d.timeout.Stop()
			}
			ctx = contextutil.Merge(ctx, d.ctx)
			conn, err := dialContext(ctx, d.xd, network, addr)
			context.AfterFunc(ctx, func() {
				defer d.mu.RUnlock()
				if d.duration > 0 {
					d.timeout.Reset(d.duration)
				}
			})
			return conn, err
		case Failed:
			defer d.mu.RUnlock()
			return nil, d.err
		case Connecting:
		default:
			panic("invalid dialer state")
		}

		d.cond.Wait()
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

func (d *Lazy) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.state != Failed {
		return
	}

	d.state, d.err = Offline, nil
	d.cond.Broadcast()
}

func (d *Lazy) Close() {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.cancel != nil {
		d.cancel(ErrCloseRequested)
	} else {
		// signal to stop any ongoing operations
		d.cond.Broadcast()
	}
}

func (d *Lazy) init() {
	d.mu.Lock()
	d.xd, d.state, d.err = nil, Connecting, nil
	d.ctx, d.cancel = context.WithCancelCause(context.Background())
	d.cond.Broadcast()
	d.mu.Unlock()

	xd, err := d.new(d.ctx)
	if err != nil {
		d.mu.Lock()
		defer d.mu.Unlock()
		d.cancel(err)
		d.xd, d.state, d.err = nil, Failed, context.Cause(d.ctx)
		d.ctx, d.cancel, d.timeout = nil, nil, nil
		d.initing.Store(false)
		d.cond.Broadcast()
		return
	}

	if w, ok := xd.(waiter); ok {
		cancel := d.cancel
		go func() {
			cancel(fmt.Errorf("underlying dialer closed: %w", w.Wait()))
		}()
	}

	d.mu.Lock()
	d.xd, d.state, d.err = xd, Online, err
	if d.duration > 0 {
		d.timeout = syncutil.NewTimeout(d.mu.RLocker(), d.duration, func() {
			if d.state != Online {
				return
			}
			d.cancel(ErrIdleTimeout)
			if c, ok := d.xd.(io.Closer); ok {
				go c.Close()
			}
			d.xd, d.state, d.err = nil, Offline, context.Cause(d.ctx)
			d.ctx, d.cancel, d.timeout = nil, nil, nil
			d.cond.Broadcast()
		})
	}
	d.initing.Store(false)
	d.cond.Broadcast()
	d.mu.Unlock()

	context.AfterFunc(d.ctx, func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		if context.Cause(d.ctx) == ErrCloseRequested {
			if c, ok := d.xd.(io.Closer); ok {
				go c.Close()
			}
		}
		d.xd, d.state, d.err = nil, Offline, context.Cause(d.ctx)
		d.ctx, d.cancel, d.timeout = nil, nil, nil
		d.cond.Broadcast()
	})

	return
}
