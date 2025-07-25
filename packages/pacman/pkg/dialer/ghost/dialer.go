package ghost

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/caseymrm/menuet"
	"golang.org/x/net/proxy"

	_ "github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/pool"
)

var localResolver = net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, network string, address string) (net.Conn, error) {
		return nil, errors.New("local resolution only")
	},
}

type Opts struct {
	Matcher HostMatcher
	Dial    func(ctx context.Context, network, address string) (net.Conn, error)
}

// Dialer directs connections based on rules.
// It supports recursive dialers.
type Dialer struct {
	matcher  HostMatcher
	fwd      func(ctx context.Context, network, address string) (net.Conn, error)
	pool     *pool.Pool[*Proxy, proxy.ContextDialer]
	resolver *net.Resolver
	app      *menuet.Application
}

// NewDialerPool initializes a pool.
func NewDialer(o Opts) *Dialer {
	d := Dialer{
		matcher: o.Matcher,
		app:     menuet.App(),
	}
	if dial := o.Dial; dial != nil {
		d.fwd = dial
	} else {
		d.fwd = (&net.Dialer{}).DialContext
	}
	d.pool = pool.New(d.factory, timeout)
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

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	proxies, found := d.matcher.MatchHost(host)

	if ips, err := localResolver.LookupIP(ctx, "ip", host); err == nil {
		ip := ips[0].String()
		if ip != host {
			address = net.JoinHostPort(ip, port)
			host = ip
			if !found {
				proxies, found = d.matcher.MatchHost(host)
			}
		}
	}

	if !found {
		slog.Debug(
			"Using forwarding dialer",
			slog.String("network", network),
			slog.String("address", address),
		)
		return d.fwd(ctx, network, address)
	}

	for _, u := range proxies {
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

func (d *Dialer) dial(p *Proxy, ctx context.Context, network, address string) (net.Conn, error) {
	dd, err := d.pool.GetCtx(ctx, p)
	if err != nil {
		return nil, err
	}

	conn, err := dd.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// factory creates a pooled dialer.
func (d *Dialer) factory(p *Proxy) (proxy.ContextDialer, error) {
	slog.Debug(
		"Creating dialer",
		slog.String("proxy", p.Redacted()),
	)

	d.app.Notification(menuet.Notification{
		Title:      "Connecting to proxy",
		Subtitle:   p.Principal(),
		Message:    "The connection to the proxy is being established.",
		Identifier: p.ID(),
	})

	dd, err := proxy.FromURL(&p.URL, d)
	if err != nil {
		d.app.Notification(menuet.Notification{
			Title:      "Proxy connection failed",
			Subtitle:   p.Principal(),
			Message:    err.Error(),
			Identifier: p.ID(),
		})
		return nil, err
	}

	xd, ok := dd.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("Dialer does not support DialContext: %s", p.Principal())
	}

	d.app.Notification(menuet.Notification{
		Title:      "Proxy connected",
		Subtitle:   p.Principal(),
		Message:    "The proxy connection has been established",
		Identifier: p.ID(),
	})

	if w, ok := dd.(interface{ Wait() error }); ok {
		go func() {
			msg := "The connection was terminated"
			if err := w.Wait(); err != nil {
				msg += err.Error()
			}
			d.app.Notification(menuet.Notification{
				Title:      "Proxy disconnected",
				Subtitle:   p.Principal(),
				Message:    msg,
				Identifier: p.ID(),
			})
		}()
	}

	return xd, nil
}

func timeout(p *Proxy) <-chan time.Time {
	if p.Query().Get("timeout") == "0" {
		return nil
	}
	return time.After(1 * time.Hour)
}
