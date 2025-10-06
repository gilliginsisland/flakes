package app

import (
	"fmt"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/caseymrm/menuet"
	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/dialer"
)

type DialerPool map[string]*PooledDialer

type PooledDialer struct {
	Label  string
	URL    *url.URL
	dialer *dialer.Lazy
	cancel func()
	state  atomic.Int32
}

func NewPooledDialer(l string, u *url.URL, fwd proxy.Dialer) *PooledDialer {
	var timeout time.Duration = 1 * time.Hour
	if t := u.Query().Get("timeout"); t != "" {
		if i, err := strconv.Atoi(t); err == nil {
			timeout = time.Duration(i) * time.Second
		}
	}
	lazy := dialer.Lazy{
		Timeout: timeout,
		New: func() (proxy.ContextDialer, error) {
			d, err := proxy.FromURL(u, fwd)
			if err != nil {
				return nil, err
			}
			xd, ok := d.(proxy.ContextDialer)
			if !ok {
				err = fmt.Errorf("Dialer does not support DialContext: %s", u.Scheme)
				return nil, err
			}
			return xd, nil
		},
	}
	pd := PooledDialer{
		Label:  l,
		URL:    u,
		dialer: &lazy,
	}
	pd.Track()
	return &pd
}

func (pd *PooledDialer) MenuItem() menuet.MenuItem {
	state := dialer.ConnectionState(pd.state.Load())

	return menuet.MenuItem{
		Text: pd.icon(state) + " " + pd.Label,
		Children: func() []menuet.MenuItem {
			var child menuet.MenuItem
			child.Text, child.Clicked = pd.action(state)
			return []menuet.MenuItem{child}
		},
	}
}

func (pd *PooledDialer) Track() {
	var ch <-chan dialer.StateSignal
	ch, pd.cancel = pd.dialer.Subscribe()
	go func() {
		for msg := range ch {
			pd.state.Store(int32(msg.State))
			menuet.App().MenuChanged()
			pd.notification(msg.State, msg.Err)
		}
	}()
}

func (pd *PooledDialer) Close() {
	pd.cancel()
	pd.dialer.Close()
}

func (pd *PooledDialer) icon(state dialer.ConnectionState) string {
	switch state {
	case dialer.Offline:
		return "⚪"
	case dialer.Online:
		return "🟢"
	case dialer.Failed:
		return "🔴"
	case dialer.Connecting:
		return "🟡"
	}
	return ""
}

func (pd *PooledDialer) action(state dialer.ConnectionState) (string, func()) {
	switch state {
	case dialer.Offline, dialer.Failed:
		return "Offline", nil
	case dialer.Online, dialer.Connecting:
		return "Disconnect", func() { pd.dialer.Close() }
	}
	return "", nil
}

func (pd *PooledDialer) notification(state dialer.ConnectionState, err error) {
	notif := menuet.Notification{
		Subtitle: pd.Label,
	}
	switch state {
	case dialer.Offline:
		notif.Title = "Proxy disconnected"
		notif.Message = "The connection was terminated."
	case dialer.Connecting:
		notif.Title = "Connecting to proxy"
		notif.Message = "The connection to the proxy is being established."
	case dialer.Online:
		notif.Title = "Proxy connected"
		notif.Message = "The proxy connection has been established."
	case dialer.Failed:
		notif.Title = "Proxy connection failed"
		notif.Message = err.Error()
	default:
		notif.Title = "Unknown connection state"
		notif.Message = "Dialer is in an unknown state."
	}
	if err != nil {
		notif.Message += " " + err.Error()
	}
	menuet.App().Notification(notif)
}
