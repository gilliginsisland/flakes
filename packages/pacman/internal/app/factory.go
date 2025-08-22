package app

import (
	"fmt"
	"net/url"

	"golang.org/x/net/proxy"

	"github.com/caseymrm/menuet"
)

func FromURL(u *url.URL, fwd proxy.Dialer) (proxy.ContextDialer, error) {
	app := menuet.App()

	app.Notification(menuet.Notification{
		Title:      "Connecting to proxy",
		Subtitle:   u.Hostname(),
		Message:    "The connection to the proxy is being established.",
		Identifier: u.Redacted(),
	})

	dd, err := proxy.FromURL(u, fwd)
	if err != nil {
		app.Notification(menuet.Notification{
			Title:      "Proxy connection failed",
			Subtitle:   u.Hostname(),
			Message:    err.Error(),
			Identifier: u.Redacted(),
		})
		return nil, err
	}

	xd, ok := dd.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("Dialer does not support DialContext: %s", u.Hostname())
	}

	app.Notification(menuet.Notification{
		Title:      "Proxy connected",
		Subtitle:   u.Hostname(),
		Message:    "The proxy connection has been established",
		Identifier: u.Redacted(),
	})

	if w, ok := dd.(interface{ Wait() error }); ok {
		go func() {
			msg := "The connection was terminated"
			if err := w.Wait(); err != nil {
				msg += err.Error()
			}
			app.Notification(menuet.Notification{
				Title:      "Proxy disconnected",
				Subtitle:   u.Hostname(),
				Message:    msg,
				Identifier: u.Redacted(),
			})
		}()
	}

	return xd, nil
}
