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
	Ruleset Ruleset
	Dial    func(ctx context.Context, network, address string) (net.Conn, error)
}

// Dialer directs connections based on rules.
// It supports recursive dialers.
type Dialer struct {
	rules    Ruleset
	fwd      func(ctx context.Context, network, address string) (net.Conn, error)
	pool     *pool.Pool[*URL, proxy.ContextDialer]
	resolver *net.Resolver
	app      *menuet.Application
}

// NewDialerPool initializes a pool.
func NewDialer(o Opts) *Dialer {
	d := Dialer{
		rules: o.Ruleset,
		app:   menuet.App(),
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

	rule := d.rules.MatchHost(host)

	if ips, err := localResolver.LookupIP(ctx, "ip", host); err == nil {
		ip := ips[0].String()
		if ip != host {
			address = net.JoinHostPort(ip, port)
			host = ip
			if rule == nil {
				rule = d.rules.MatchHost(host)
			}
		}
	}

	if rule == nil || len(rule.Proxies) == 0 {
		slog.Debug(
			"Using forwarding dialer",
			slog.String("network", network),
			slog.String("address", address),
		)
		return d.fwd(ctx, network, address)
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
	dd, err := d.pool.GetCtx(ctx, u)
	if err != nil {
		d.app.Notification(menuet.Notification{
			Title:    "Proxy connection failed",
			Subtitle: u.Redacted(),
			Message:  err.Error(),
		})
		return nil, err
	}

	conn, err := dd.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// factory creates a pooled dialer.
func (d *Dialer) factory(u *URL) (proxy.ContextDialer, error) {
	slog.Debug(
		"Creating dialer",
		slog.String("proxy", u.Redacted()),
	)

	d.app.Notification(menuet.Notification{
		Title:    "Connecting to proxy",
		Subtitle: u.Redacted(),
		Message:  "The connection to the proxy is being established.",
	})

	dd, err := proxy.FromURL(&u.URL, d)
	if err != nil {
		d.app.Notification(menuet.Notification{
			Title:    "Proxy connection failed",
			Subtitle: u.Redacted(),
			Message:  err.Error(),
		})
		return nil, err
	}

	xd, ok := dd.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("Dialer does not support DialContext: %s", u.Redacted())
	}

	d.app.Notification(menuet.Notification{
		Title:    "Proxy connected",
		Subtitle: u.Redacted(),
		Message:  "The proxy connection has been established",
	})

	if w, ok := dd.(interface{ Wait() error }); ok {
		go func() {
			msg := "The connection was terminated"
			if err := w.Wait(); err != nil {
				msg += err.Error()
			}
			d.app.Notification(menuet.Notification{
				Title:    "Proxy disconnected",
				Subtitle: u.Redacted(),
				Message:  msg,
			})
		}()
	}

	return xd, nil
}

func timeout(u *URL) <-chan time.Time {
	if u.Query().Get("timeout") == "0" {
		return nil
	}
	return time.After(1 * time.Hour)
}
