package dialer

import (
	"context"
	"fmt"
	"net"

	"github.com/gilliginsisland/pacman/internal/netutil"
	"golang.org/x/net/proxy"
)

// GHost directs connections based on rules.
// It supports recursive dialers.
type GHost struct {
	Ruleset *Ruleset
}

func NewGHost() *GHost {
	g := &GHost{}
	g.Ruleset = NewRuleset(g)
	return g
}

func (g *GHost) Dial(network, address string) (net.Conn, error) {
	return g.DialContext(nil, network, address)
}

func (g *GHost) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	dialers, err := g.dialersForAddress(address)
	if err != nil {
		return nil, err
	}

	for _, d := range dialers {
		conn, err := netutil.DialContext(ctx, d, network, address)
		if err != nil {
			continue
		}
		return conn, nil
	}

	return nil, fmt.Errorf("all dialers failed for %s", address)
}

func (g *GHost) dialersForAddress(address string) ([]proxy.Dialer, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	for _, r := range g.Ruleset.rules {
		for _, m := range r.matchers {
			if m.MatchString(host) {
				return r.dialers, nil
			}
		}
	}

	return []proxy.Dialer{proxy.Direct}, nil
}
