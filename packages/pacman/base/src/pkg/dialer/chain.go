package dialer

import (
	"context"
	"errors"
	"net"

	"golang.org/x/net/proxy"
)

var _ proxy.ContextDialer = (Chain)(nil)

type Chain []proxy.ContextDialer

func (c Chain) Dial(network, address string) (net.Conn, error) {
	return c.DialContext(context.Background(), network, address)
}

func (c Chain) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	var errs []error
	for _, d := range c {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if conn, err := d.DialContext(ctx, network, address); err == nil {
			return conn, nil
		} else {
			errs = append(errs, err)
		}
	}
	return nil, errors.Join(errs...)
}
