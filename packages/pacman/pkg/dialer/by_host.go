package dialer

import (
	"context"
	"errors"
	"net"

	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/trie"
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
	trie    trie.Trie[proxy.ContextDialer]
}

// Add parses a string specifying a host that should use the given proxy.
// Each value is either an IP address, a CIDR range, a zone (*.example.com) or a
// host name (example.com).
func (d *ByHost) Add(host string, p proxy.ContextDialer) {
	d.trie.Insert(host, p)
}

func (d *ByHost) Hosts(yield func(string) bool) {
	for k := range d.trie.Walk {
		if !yield(k) {
			return
		}
	}
}

func (d *ByHost) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *ByHost) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	pd, ok := d.trie.Match(host)
	// Check local resolver in case of /etc/hosts ip override
	if ips, err := resolver.LookupIP(ctx, "ip", host); err == nil && len(ips) > 0 {
		ip := ips[0].String()
		if ip != host {
			// If different then override host port
			address, host = net.JoinHostPort(ip, port), ip
			// If there was no hostname rule there may be an ip rule
			if !ok {
				pd, ok = d.trie.Match(host)
			}
		}
	}
	if pd == nil {
		pd = d.Default
	}

	return pd.DialContext(ctx, network, address)
}
