package app

import (
	"context"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"time"
	"unsafe"

	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/menuet/appkit"
	"github.com/gilliginsisland/pacman/pkg/menuet/core"
	"github.com/gilliginsisland/pacman/pkg/menuet/notifications"
	"github.com/gilliginsisland/pacman/pkg/notify"
)

var failureNotificationCategory = notifications.NewCategory(notifications.CategoryContent{
	Identifier: "proxy-failed",
	Actions: []*notifications.Action{
		notifications.NewAction(notifications.ActionContent{
			Identifier: "proxy-reset",
			Title:      "Reset",
		}),
	},
	Options: notifications.CategoryOptionCustomDismiss,
})

type DialerPool struct {
	dialers  map[string]*PooledDialer
	fwd      proxy.Dialer
	callback *core.Callback
}

func (pool *DialerPool) MenuItems() []*appkit.MenuItem {
	if len(pool.dialers) == 0 {
		return []*appkit.MenuItem{
			appkit.NewMenuItem().SetTitle("No Proxies Configured"),
		}
	}
	items := make([]*appkit.MenuItem, 0, len(pool.dialers))
	for _, label := range slices.Sorted(maps.Keys(pool.dialers)) {
		items = append(items, pool.dialers[label].menu.MenuItem)
	}
	return items
}

type PooledDialer struct {
	Label   string
	URL     *url.URL
	updates chan dialer.StateUpdate

	dialer *dialer.Lazy
	menu   struct {
		*appkit.MenuItem
		*appkit.Menu
		input struct {
			*appkit.MenuItem
			header *appkit.MenuItem
			view   *appkit.InputView
		}
		connect    *appkit.MenuItem
		cancel     *appkit.MenuItem
		disconnect *appkit.MenuItem
		reset      *appkit.MenuItem
	}
}

func (pool *DialerPool) NewPooledDialer(l string, u *url.URL) *PooledDialer {
	var timeout time.Duration = 1 * time.Hour
	if t := u.Query().Get("timeout"); t != "" {
		if i, err := strconv.Atoi(t); err == nil {
			timeout = time.Duration(i) * time.Second
		}
	}

	pd := PooledDialer{
		Label:   l,
		URL:     u,
		updates: make(chan dialer.StateUpdate),
	}
	core.NewString(l).WithPointer(func(obj unsafe.Pointer) {
		pd.menu.MenuItem = appkit.NewMenuItem().
			SetTag(int(actionDialer)).
			SetRepresentedObject(obj)
		// Keep each action's identity fixed, including for queued clicks.
		pd.menu.connect = appkit.NewMenuItem().
			SetTitle("Connect").
			SetTag(int(actionConnect)).
			SetRepresentedObject(obj).
			SetCallback(pool.callback)
		pd.menu.cancel = appkit.NewMenuItem().
			SetTitle("Cancel").
			SetTag(int(actionCancel)).
			SetRepresentedObject(obj).
			SetCallback(pool.callback)
		pd.menu.disconnect = appkit.NewMenuItem().
			SetTitle("Disconnect").
			SetTag(int(actionDisconnect)).
			SetRepresentedObject(obj).
			SetCallback(pool.callback)
		pd.menu.reset = appkit.NewMenuItem().
			SetTitle("Reset").
			SetTag(int(actionReset)).
			SetRepresentedObject(obj).
			SetCallback(pool.callback)
	})
	pd.menu.input.view = appkit.NewInputView("Enter authentication input")
	pd.menu.input.header = appkit.NewMenuItemSectionHeader("").
		SetHidden(true)
	pd.menu.input.MenuItem = appkit.NewMenuItem().
		SetHidden(true)
	pd.menu.input.view.WithPointer(func(obj unsafe.Pointer) {
		pd.menu.input.SetView(obj)
	})
	pd.menu.Menu = appkit.NewMenu().
		Append(pd.menu.input.header, pd.menu.input.MenuItem, pd.menu.connect, pd.menu.cancel, pd.menu.disconnect, pd.menu.reset)
	pd.menu.SetSubmenu(pd.menu.Menu)
	pd.updateMenu(dialer.Offline)
	pd.dialer = dialer.NewLazy(
		func(ctx context.Context) (dialer.Dialer, error) {
			// This bounds shared tunnel setup, not the returned tunnel's lifetime.
			ctx, cancel := context.WithTimeout(context.WithValue(ctx, "label", l), 2*time.Minute)
			defer cancel()
			return dialer.FromURLContext(ctx, u, pool.fwd)
		},
		dialer.LazyOptions{
			IdleTimeout: timeout,
			Updates:     pd.updates,
		},
	)
	go pd.run()
	return &pd
}

func (pd *PooledDialer) run() {
	state := dialer.Offline
	for {
		var update dialer.StateUpdate
		select {
		case update = <-pd.updates:
		case <-pd.dialer.Done():
			return
		}
		if update.State == state {
			continue
		}
		pd.updateMenu(update.State)
		if update.State != dialer.Offline || state == dialer.Online {
			pd.notification(update.State, update.Err)
		}
		state = update.State
	}
}

func (pd *PooledDialer) updateMenu(state dialer.ConnectionState) {
	pd.menu.SetTitle(pd.icon(state) + " " + pd.Label).
		SetBadge(state.String())
	pd.menu.connect.SetHidden(state != dialer.Offline)
	pd.menu.cancel.SetHidden(state != dialer.Connecting)
	pd.menu.disconnect.SetHidden(state != dialer.Online)
	pd.menu.reset.SetHidden(state != dialer.Failed)
}

func (pd *PooledDialer) Close() {
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

func (pd *PooledDialer) notification(state dialer.ConnectionState, err error) {
	notif := notify.Notification{
		Identifier:          "status:" + pd.Label,
		Title:               pd.Label,
		PresentationOptions: notifications.PresentationOptionBanner,
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
		notif.Body = ""
		if err != nil {
			notif.CategoryIdentifier = "proxy-failed"
		}
	default:
		notif.Subtitle = "Unknown connection state"
		notif.Body = "Dialer is in an unknown state."
	}
	if err != nil {
		if notif.Body != "" {
			notif.Body += "\n"
		}
		notif.Body += err.Error()
	}
	switch notif.CategoryIdentifier {
	case "proxy-failed":
		responses, cleanup := notify.NotifyCh(notif)
		go func() {
			defer cleanup()
			select {
			case response := <-responses:
				if response.ActionIdentifier == "proxy-reset" {
					pd.dialer.Reset(dialer.Reset)
				}
			case <-pd.dialer.Done():
			}
		}()
	default:
		notify.Notify(notif)
	}
}
