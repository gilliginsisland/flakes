package app

import (
	"context"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/iterutil"
	"github.com/gilliginsisland/pacman/pkg/menuet"
	"github.com/gilliginsisland/pacman/pkg/notify"
)

type DialerPool map[string]*PooledDialer

func (dp DialerPool) MenuItems() []menuet.Itemer {
	items := make([]menuet.Itemer, len(dp))
	var idx int
	for _, pd := range iterutil.SortedMapIter(dp) {
		items[idx] = &pd.menu
		idx++
	}
	return items
}

type PooledDialer struct {
	Label  string
	URL    *url.URL
	ctx    context.Context
	cancel func()
	dialer *dialer.Lazy
	state  atomic.Int32
	menu   menuet.MenuItem
	child  menuet.MenuItem
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
	pd.menu.Submenu = &pd.menu
	return &pd
}

func (pd *PooledDialer) updateMenu(state dialer.ConnectionState) {
	pd.menu.Text = pd.icon(state) + " " + pd.Label
	pd.child.Text, pd.child.Clicked = pd.action(state)
}

func (pd *PooledDialer) Close() {
	pd.cancel()
	pd.dialer.Close()
}

func (pd *PooledDialer) Track(cb func()) {
	for state, err := range pd.dialer.Subscribe {
		pd.updateMenu(state)
		cb()
		pd.notification(state, err)
		if pd.ctx.Err() != nil {
			break
		}
	}
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
		Title: pd.Label,
	}
	switch state {
	case dialer.Offline:
		notif.Subtitle = "Proxy disconnected"
		notif.Body = "The connection was terminated."
	case dialer.Connecting:
		notif.Subtitle = "Connecting to proxy"
		notif.Body = "The connection to the proxy is being established."
	case dialer.Online:
		notif.Subtitle = "Proxy connected"
		notif.Body = "The proxy connection has been established."
	case dialer.Failed:
		notif.Subtitle = "Proxy connection failed"
		notif.Body = err.Error()
	default:
		notif.Subtitle = "Unknown connection state"
		notif.Body = "Dialer is in an unknown state."
	}
	if err != nil {
		notif.Body += " " + err.Error()
	}
	notify.Notify(notif)
}
