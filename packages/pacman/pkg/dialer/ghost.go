package dialer

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/gilliginsisland/pacman/internal/syncutil"
	"golang.org/x/net/proxy"
)

// GHost directs connections based on rules.
// It supports recursive dialers.
type GHost struct {
	rules   Ruleset
	forward proxy.ContextDialer
	dialers *syncutil.Pool[*URL, *dialer]
}

// NewDialerPool initializes a pool.
func NewGHost(rules Ruleset, forward proxy.ContextDialer) *GHost {
	if forward == nil {
		forward = proxy.Direct
	}
	g := GHost{
		rules:   rules,
		forward: forward,
	}
	g.dialers = syncutil.NewPool(g.newDialer)
	return &g
}

func (g *GHost) Dial(network, address string) (net.Conn, error) {
	return g.DialContext(context.Background(), network, address)
}

func (g *GHost) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	slog.Debug(
		"DialingContext",
		slog.String("network", network),
		slog.String("address", address),
	)

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	rule := g.rules.MatchHost(host)
	if rule == nil || len(rule.Proxies) == 0 {
		slog.Debug(
			"Using forwarding dialer",
			slog.String("network", network),
			slog.String("address", address),
		)
		return g.forward.DialContext(ctx, network, address)
	}

	for _, u := range rule.Proxies {
		if err := ctx.Err(); err != nil {
			slog.Debug(
				"Aborting due to context cancelled",
				slog.String("network", network),
				slog.String("address", address),
			)
			return nil, err
		}

		conn, err := g.dial(u, ctx, network, address)
		if err != nil {
			continue
		}

		return conn, nil
	}

	return nil, fmt.Errorf("all dialers failed for %s", address)
}

func (g *GHost) dial(u *URL, ctx context.Context, network, address string) (net.Conn, error) {
	d, err := g.dialers.Get(u)
	if err != nil {
		slog.Warn(
			"skipping unsupported proxy",
			slog.Any("proxy", u),
			slog.Any("error", err),
			slog.String("network", network),
			slog.String("address", address),
		)
		return nil, err
	}

	slog.Debug(
		"Trying proxy dialer",
		slog.Any("proxy", u),
		slog.String("network", network),
		slog.String("address", address),
	)
	conn, err := d.DialContext(ctx, network, address)
	if err != nil {
		slog.Error(
			"proxy connection failed",
			slog.Any("proxy", u),
			slog.Any("error", err),
			slog.String("network", network),
			slog.String("address", address),
		)
		return nil, err
	}

	// reference counting only applies if the underlying dialer
	// supports being closed.
	if _, ok := d.ContextDialer.(io.Closer); ok {
		go func() {
			d.ref <- 1
			<-ctx.Done()
			d.ref <- -1
		}()
	}

	return conn, nil
}
