package dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"sync"

	"golang.org/x/net/proxy"
)

// ctxSchemes is a map from URL schemes to a function that creates a Dialer
// from a URL with such a scheme.
var ctxSchemes map[string]func(context.Context, *url.URL, proxy.Dialer) (Dialer, error)

// RegisterContextDialerType takes a URL scheme and a function to generate Dialers from
// a URL with that scheme and a forwarding Dialer. Registered schemes are used
// by FromURL.
func RegisterContextDialerType(scheme string, fn func(context.Context, *url.URL, proxy.Dialer) (Dialer, error)) {
	if ctxSchemes == nil {
		ctxSchemes = make(map[string]func(context.Context, *url.URL, proxy.Dialer) (Dialer, error))
	}
	ctxSchemes[scheme] = fn

	// Register background-compatible version with the standard proxy registry.
	proxy.RegisterDialerType(scheme, func(u *url.URL, forward proxy.Dialer) (proxy.Dialer, error) {
		return fn(context.Background(), u, forward)
	})
}

// FromURLContext behaves like proxy.FromURL but supports context and cancellation.
//
// If the scheme was registered with RegisterContextDialerType, it uses that.
// Otherwise, it falls back to proxy.FromURL in a goroutine so that
// ctx cancellation can return immediately even if setup is blocking.
func FromURLContext(ctx context.Context, u *url.URL, forward proxy.Dialer) (Dialer, error) {
	if fn := ctxSchemes[u.Scheme]; fn != nil {
		return fn(ctx, u, forward)
	}
	var (
		dialer proxy.Dialer
		err    error
		done   = make(chan struct{}, 1)
	)
	go func() {
		dialer, err = proxy.FromURL(u, forward)
		close(done)
	}()
	select {
	case <-ctx.Done():
		go func() {
			<-done
			if c, ok := dialer.(io.Closer); ok {
				c.Close()
			}
		}()
		return nil, ctx.Err()
	case <-done:
		if err != nil {
			return nil, err
		}
		if d, ok := dialer.(Dialer); ok {
			return d, err
		}
		return newForwardDialer(dialer), nil
	}
}

// forwardDialer preserves native lifecycle methods and supplies missing ones.
type forwardDialer struct {
	proxy.Dialer
	xd     proxy.ContextDialer
	closer io.Closer
	waiter interface{ Wait() error }

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
	err    error

	mu    sync.Mutex
	conns map[net.Conn]struct{}
}

func newForwardDialer(dialer proxy.Dialer) Dialer {
	d := forwardDialer{Dialer: dialer}
	d.xd, _ = dialer.(proxy.ContextDialer)
	d.closer, _ = dialer.(io.Closer)
	d.waiter, _ = dialer.(interface{ Wait() error })
	if d.closer == nil {
		// Only a backend without Close needs shim-owned cancellation.
		d.ctx, d.cancel = context.WithCancel(context.Background())
		d.conns = make(map[net.Conn]struct{})
	}
	if d.waiter == nil {
		d.done = make(chan struct{})
	}
	return &d
}

func (d *forwardDialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *forwardDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if d.ctx != nil {
		if err := d.ctx.Err(); err != nil {
			return nil, err
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		stop := context.AfterFunc(d.ctx, cancel)
		defer cancel()
		defer stop()
	}
	var conn net.Conn
	var err error
	if d.xd != nil {
		conn, err = d.xd.DialContext(ctx, network, address)
	} else {
		conn, err = dialContext(ctx, d.Dialer, network, address)
	}
	if err != nil {
		return nil, err
	}
	// A native Close owns cancellation of in-progress dials and established
	// connections. Only backends without it use the shim's connection registry.
	if d.closer != nil {
		return conn, nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.ctx.Err(); err != nil {
		// Close won before this successful dial could be registered.
		conn.Close()
		return nil, err
	}
	d.conns[conn] = struct{}{}
	return &trackedConn{
		Conn: conn,
		onClose: func() {
			d.mu.Lock()
			defer d.mu.Unlock()
			delete(d.conns, conn)
		},
	}, nil
}

func (d *forwardDialer) Wait() error {
	if d.waiter != nil {
		return d.waiter.Wait()
	}
	// Without a native waiter, only explicit Close ends the shim's lifetime;
	// individual connection failures do not mean the shared dialer has died.
	<-d.done
	return nil
}

func (d *forwardDialer) Close() error {
	d.once.Do(func() {
		if d.closer != nil {
			d.err = d.closer.Close()
		} else {
			d.cancel()
			d.mu.Lock()
			conns := d.conns
			d.conns = nil
			d.mu.Unlock()
			// Close outside mu: recursive proxy connections may run close hooks
			// that acquire their owner's connection-registry mutex.
			for conn := range conns {
				d.err = errors.Join(d.err, conn.Close())
			}
		}
		if d.done != nil {
			close(d.done)
		}
	})
	return d.err
}
