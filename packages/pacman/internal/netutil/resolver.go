package netutil

import (
	"context"
	"errors"
	"net"
	"strings"
)

// contextKey is used to store/retrieve values from context
// without key collisions.
type contextKey string

const dnsHostKey contextKey = "dnshost"

// Resolver is a one-method interface for DNS lookups.
type IPLookuper interface {
	LookupIP(ctx context.Context, network, host string) ([]net.IP, error)
}

// resolver implements Resolver and wraps net.Resolver
// to capture and rewrite DNS errors with the real server.
type resolver struct {
	net.Resolver
}

// NewResolver returns a net.Resolver using the specified DNS servers.
// It takes a dial function matching net.Resolver.Dial. If nil, net.DialContext is used.
func NewResolver(servers []string, dial func(ctx context.Context, network, address string) (net.Conn, error)) IPLookuper {
	if dial == nil {
		var d net.Dialer
		dial = d.DialContext
	}

	return &resolver{
		Resolver: net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, host string) (net.Conn, error) {
				// Track which DNS server was attempted.
				if ptr, _ := ctx.Value(dnsHostKey).(*string); ptr != nil {
					*ptr = host
				}

				var errs []error

				for _, addr := range servers {
					conn, err := dial(ctx, network, addr)
					if err == nil {
						return conn, nil
					}
					errs = append(errs, err)
				}

				if len(servers) == 0 {
					return nil, errors.New("no DNS servers configured")
				}

				return nil, errors.Join(errs...)
			},
		},
	}
}

// LookupIP resolves the host using the given context and network ("ip", "ip4", or "ip6").
func (r *resolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	var attempted string
	ctx = context.WithValue(ctx, dnsHostKey, &attempted)

	ips, err := r.Resolver.LookupIP(ctx, network, host)
	if err != nil && attempted != "" {
		return nil, errors.New(strings.ReplaceAll(err.Error(), attempted, "x.x.x.x"))
	}

	return ips, nil
}
