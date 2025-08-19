package dialer

import (
	"context"
	"errors"
	"net"

	"golang.org/x/net/proxy"
)

var resolver = net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, network string, address string) (net.Conn, error) {
		return nil, errors.New("local resolution only")
	},
}

// ByHost directs connections based on rules.
// It supports recursive dialers.
type ByHost func(hostname string) proxy.ContextDialer

func (d ByHost) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d ByHost) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	pd := d(host)

	if ips, err := resolver.LookupIP(ctx, "ip", host); err == nil {
		ip := ips[0].String()
		if ip != host {
			address = net.JoinHostPort(ip, port)
			host = ip
			if pd == nil {
				pd = d(host)
			}
		}
	}

	if pd == nil {
		pd = &net.Dialer{}
	}

	return pd.DialContext(ctx, network, address)
}
