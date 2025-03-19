package netutil

import (
	"context"
	"net"

	"golang.org/x/net/proxy"
)

// Works like DialContext on net.Dialer but using the passed dialer.
//
// The passed ctx is only used for returning the Conn, not the lifetime of the Conn.
//
// Dialers that do not implement ContextDialer can leak a goroutine for as long as it
// takes the underlying Dialer implementation to timeout.
//
// A Conn returned from a successful Dial after the context has been cancelled will be immediately closed.
func DialContext(ctx context.Context, d proxy.Dialer, network, address string) (net.Conn, error) {
	if d == nil {
		d = proxy.Direct
	}

	if ctx == nil {
		return d.Dial(network, address)
	}

	if xd, ok := d.(proxy.ContextDialer); ok {
		return xd.DialContext(ctx, network, address)
	}

	var (
		conn net.Conn
		done = make(chan net.Conn, 1)
		err  error
	)
	go func() {
		conn, err = d.Dial(network, address)
		close(done)
		if conn != nil && ctx.Err() != nil {
			conn.Close()
		}
	}()
	select {
	case <-ctx.Done():
		err = ctx.Err()
	case <-done:
	}
	return conn, err
}
