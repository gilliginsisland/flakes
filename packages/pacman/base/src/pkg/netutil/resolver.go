package netutil

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
)

// Resolver is a one-method interface for DNS lookups.
type IPLookuper interface {
	LookupIP(ctx context.Context, network, host string) ([]net.IP, error)
}

// resolver masks Go's nominal DNS server, which the custom dialer overrides.
type resolver struct {
	net.Resolver
}

// NewResolver returns a net.Resolver using the specified DNS servers.
// Each DNS dial selects a server uniformly at random.
// It takes a dial function matching net.Resolver.Dial. If nil, net.DialContext is used.
func NewResolver(servers []string, dial func(ctx context.Context, network, address string) (net.Conn, error)) IPLookuper {
	if dial == nil {
		var d net.Dialer
		dial = d.DialContext
	}

	return &resolver{
		Resolver: net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				if len(servers) == 0 {
					return nil, errors.New("no DNS servers configured")
				}
				return dial(ctx, network, servers[rand.IntN(len(servers))])
			},
		},
	}
}

// LookupIP resolves the host using the given context and network ("ip", "ip4", or "ip6").
func (r *resolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	ips, err := r.Resolver.LookupIP(ctx, network, host)
	if dnsErr, ok := errors.AsType[*net.DNSError](err); ok && dnsErr.Server != "" {
		// Lookups can share errors, so leave the original unchanged.
		rewritten := *dnsErr
		rewritten.Server = "x.x.x.x"
		return nil, &rewritten
	}
	return ips, err
}
