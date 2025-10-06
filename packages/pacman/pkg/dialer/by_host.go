package dialer

import (
	"context"
	"errors"
	"net"
	"sync/atomic"

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
type ByHost struct {
	Default proxy.ContextDialer
	rs      atomic.Pointer[RuleSet]
}

func (d *ByHost) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *ByHost) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	rs := d.rs.Load()
	pd, ok := rs.Match(host)
	// Check local resolver in case of /etc/hosts ip override
	if ips, err := resolver.LookupIP(ctx, "ip", host); err == nil && len(ips) > 0 {
		ip := ips[0].String()
		if ip != host {
			// If different then override host port
			address, host = net.JoinHostPort(ip, port), ip
			// If there was no hostname rule there may be an ip rule
			if !ok {
				pd, ok = rs.Match(host)
			}
		}
	}
	if pd == nil {
		if d.Default != nil {
			pd = d.Default
		} else {
			pd = proxy.Direct
		}
	}

	return pd.DialContext(ctx, network, address)
}

// Swap installs a new ruleset atomically.
// If newRS == nil, an empty RuleSet is installed.
func (d *ByHost) Swap(rs *RuleSet) {
	d.rs.Store(rs)
}

func (d *ByHost) Hosts(yield func(string) bool) {
	if rs := d.rs.Load(); rs != nil {
		rs.Hosts(yield)
	}
}
