package ghost

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/gilliginsisland/pacman/internal/syncutil"
	"golang.org/x/net/proxy"

	_ "github.com/gilliginsisland/pacman/pkg/dialer"
)

// Dialer directs connections based on rules.
// It supports recursive dialers.
type Dialer struct {
	rules   Ruleset
	forward proxy.ContextDialer
	dialers *syncutil.Pool[*URL, *refDialer]
}

// NewDialerPool initializes a pool.
func NewDialer(rules Ruleset, forward proxy.ContextDialer) *Dialer {
	if forward == nil {
		forward = proxy.Direct
	}
	d := Dialer{
		rules:   rules,
		forward: forward,
	}
	d.dialers = syncutil.NewPool(d.newDialer)
	return &d
}

func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	slog.Debug(
		"DialingContext",
		slog.String("network", network),
		slog.String("address", address),
	)

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	rule := d.rules.MatchHost(host)
	if rule == nil || len(rule.Proxies) == 0 {
		slog.Debug(
			"Using forwarding dialer",
			slog.String("network", network),
			slog.String("address", address),
		)
		return d.forward.DialContext(ctx, network, address)
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

		slog.Debug(
			"trying proxy dialer",
			slog.String("proxy", u.Redacted()),
			slog.String("network", network),
			slog.String("address", address),
		)
		conn, err := d.dial(u, ctx, network, address)
		if err != nil {
			slog.Error(
				"proxy connection failed",
				slog.String("proxy", u.Redacted()),
				slog.Any("error", err),
				slog.String("network", network),
				slog.String("address", address),
			)
			continue
		}

		return conn, nil
	}

	return nil, fmt.Errorf("all dialers failed for %s", address)
}

func (d *Dialer) dial(u *URL, ctx context.Context, network, address string) (net.Conn, error) {
	dd, err := d.dialers.Get(u)
	if err != nil {
		return nil, err
	}

	if dd.ref != nil {
		go func() {
			dd.ref <- 1
			<-ctx.Done()
			dd.ref <- -1
		}()
	}

	conn, err := dd.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
