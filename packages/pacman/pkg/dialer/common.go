package dialer

import (
	"context"
	"net"

	"golang.org/x/net/proxy"
)

// WARNING: this can leak a goroutine for as long as the underlying Dialer implementation takes to timeout
// A Conn returned from a successful Dial after the context has been cancelled will be immediately closed.
func dialContext(ctx context.Context, d proxy.Dialer, network, address string) (net.Conn, error) {
	if xd, ok := d.(proxy.ContextDialer); ok {
		return xd.DialContext(ctx, network, address)
	}
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
