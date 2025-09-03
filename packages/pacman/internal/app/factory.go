package app

import (
	"fmt"
	"log/slog"
	"net/url"

	"golang.org/x/net/proxy"

	"github.com/caseymrm/menuet"
	"github.com/gilliginsisland/pacman/pkg/dialer"
)

func FromURL(u *url.URL, fwd proxy.Dialer, onState func(dialer.ConnectionState)) (proxy.ContextDialer, error) {
	slog.Debug(
		"Creating dialer",
		slog.String("proxy", u.Redacted()),
	)

	app := menuet.App()

	go onState(dialer.Connecting)
	go app.Notification(menuet.Notification{
		Title:    "Connecting to proxy",
		Subtitle: u.Hostname(),
		Message:  "The connection to the proxy is being established.",
	})

	dd, err := proxy.FromURL(u, fwd)
	if err != nil {
		go onState(dialer.Failed)
		go app.Notification(menuet.Notification{
			Title:    "Proxy connection failed",
			Subtitle: u.Hostname(),
			Message:  err.Error(),
		})
		return nil, err
	}

	xd, ok := dd.(proxy.ContextDialer)
	if !ok {
		go onState(dialer.Failed)
		return nil, fmt.Errorf("Dialer does not support DialContext: %s", u.Hostname())
	}

	go onState(dialer.Online)
	go app.Notification(menuet.Notification{
		Title:    "Proxy connected",
		Subtitle: u.Hostname(),
		Message:  "The proxy connection has been established",
	})

	if w, ok := dd.(interface{ Wait() error }); ok {
		go func() {
			msg := "The connection was terminated"
			if err := w.Wait(); err != nil {
				msg += err.Error()
			}
			go onState(dialer.Offline)
			go app.Notification(menuet.Notification{
				Title:    "Proxy disconnected",
				Subtitle: u.Hostname(),
				Message:  msg,
			})
		}()
	}

	return xd, nil
}
