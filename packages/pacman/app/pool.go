package app

import (
	"context"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/menuet"
	"github.com/gilliginsisland/pacman/pkg/notify"
)

type DialerPool map[string]*PooledDialer

type PooledDialer struct {
	Label  string
	URL    *url.URL
	ctx    context.Context
	cancel func()
	dialer *dialer.Lazy
	state  atomic.Int32
}

func NewPooledDialer(l string, u *url.URL, fwd proxy.Dialer) *PooledDialer {
	var timeout time.Duration = 1 * time.Hour
	if t := u.Query().Get("timeout"); t != "" {
		if i, err := strconv.Atoi(t); err == nil {
			timeout = time.Duration(i) * time.Second
		}
	}
	pd := PooledDialer{
		Label: l,
		URL:   u,
		dialer: dialer.NewLazy(func(ctx context.Context) (proxy.Dialer, error) {
			ctx, cancel := context.WithCancelCause(
				context.WithValue(ctx, "label", l),
			)
			defer time.AfterFunc(2*time.Minute, func() {
				cancel(context.DeadlineExceeded)
			}).Stop()
			return dialer.FromURLContext(ctx, u, fwd)
		}, timeout),
	}
	pd.ctx, pd.cancel = context.WithCancel(context.Background())
	go func() {
		for state, err := range pd.dialer.Subscribe {
			if pd.state.Swap(int32(state)) != int32(state) {
				menuet.App().MenuChanged()
				pd.notification(state, err)
			}
			if pd.ctx.Err() != nil {
				break
			}
		}
	}()
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
	case dialer.Offline:
		return "Offline", nil
	case dialer.Failed:
		return "Reset", pd.dialer.Reset
	case dialer.Online, dialer.Connecting:
		return "Disconnect", pd.dialer.Close
	}
	return "", nil
}

func (pd *PooledDialer) notification(state dialer.ConnectionState, err error) {
	notif := notify.Notification{
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
	notify.Notify(notif)
}
