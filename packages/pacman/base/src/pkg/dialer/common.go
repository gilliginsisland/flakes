package dialer

import (
	"context"
	"net"

	"golang.org/x/net/proxy"
)

type trackedConn struct {
	net.Conn
	// Close can be repeated; callers supply an idempotent hook when needed.
	onClose func()
}

func (conn *trackedConn) Close() error {
	defer conn.onClose()
	return conn.Conn.Close()
}

// DialContext uses a context-aware dialer when available.
// WARNING: this can leak a goroutine for as long as the underlying Dialer implementation takes to timeout
// A Conn returned from a successful Dial after the context has been cancelled will be immediately closed.
func DialContext(ctx context.Context, d proxy.Dialer, network, address string) (net.Conn, error) {
	if xd, ok := d.(proxy.ContextDialer); ok {
		return xd.DialContext(ctx, network, address)
	}
	return dialContext(ctx, d, network, address)
}

func dialContext(ctx context.Context, d proxy.Dialer, network, address string) (net.Conn, error) {
	var (
		conn net.Conn
		done = make(chan struct{})
		err  error
	)
	go func() {
		conn, err = d.Dial(network, address)
		close(done)
	}()
	select {
	case <-ctx.Done():
		go func() {
			<-done
			if conn != nil {
				conn.Close()
			}
		}()
		return nil, ctx.Err()
	case <-done:
		return conn, err
	}
}
