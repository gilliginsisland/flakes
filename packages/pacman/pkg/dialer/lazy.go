package dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gilliginsisland/pacman/pkg/contextutil"
	"golang.org/x/net/proxy"
)

var (
	ErrUnderlyingClosed = errors.New("underlying dialer closed")
	ErrCloseRequested   = errors.New("close requested")
	ErrIdleTimeout      = errors.New("idle timeout reached")
)

type ConnectionState int

const (
	Offline ConnectionState = iota
	Connecting
	Failed
	Online
)

type Lazy struct {
	Timeout time.Duration
	New     func() (proxy.ContextDialer, error)

	proxy.ContextDialer

	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelCauseFunc
	timer     *time.Timer
	timerRace atomic.Bool
}

func (d *Lazy) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	d.mu.RLock()

	for d.ContextDialer == nil {
		d.mu.RUnlock()
		if err := d.init(); err != nil {
			return nil, err
		}
		d.mu.RLock()
	}

	if d.timer != nil {
		if !d.timer.Stop() {
			d.timerRace.Store(true)
		}
	}

	ctx = contextutil.Merge(ctx, d.ctx)
	conn, err := d.ContextDialer.DialContext(ctx, network, addr)
	context.AfterFunc(ctx, func() {
		d.timer.Reset(d.Timeout)
		d.mu.RUnlock()
	})

	return conn, err
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

func (d *Lazy) init() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.ContextDialer != nil {
		return nil
	}

	xd, err := d.New()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	timer := time.NewTimer(d.Timeout)

	var tc <-chan time.Time
	if d.Timeout > 0 {
		tc = timer.C
	}

	d.ContextDialer = xd
	d.ctx, d.cancel = ctx, cancel
	d.timer = timer
	d.timerRace.Store(false)

	if w, ok := xd.(interface{ Wait() error }); ok {
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

		switch context.Cause(ctx) {
		case ErrCloseRequested, ErrIdleTimeout:
			if c, ok := xd.(io.Closer); ok {
				go c.Close()
			}
		}

		d.ContextDialer = nil
		d.ctx, d.cancel = nil, nil
		d.timer = nil
		d.timerRace.Store(false)
	}()

	return nil
}
