package app

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"sync/atomic"
	"time"
	"weak"

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

var authFailureNotificationCategory = notifications.NewCategory(notifications.CategoryContent{
	Identifier: "proxy-auth-failed",
	Actions: []*notifications.Action{
		notifications.NewAction(notifications.ActionContent{Identifier: "proxy-reset", Title: "Reset"}),
		notifications.NewAction(notifications.ActionContent{Identifier: "proxy-edit-account", Title: "Edit Account…"}),
	},
	Options: notifications.CategoryOptionCustomDismiss,
})

var statusNotificationCounter atomic.Uint64

type DialerPool struct {
	dialers   map[string]*PooledDialer
	fwd       proxy.Dialer
	callback  *core.Callback
	animation *statusAnimation
}

func (pool *DialerPool) MenuItems() []*appkit.MenuItem {
	if len(pool.dialers) == 0 {
		return []*appkit.MenuItem{
			appkit.NewMenuItem().SetTitle("No Proxies Configured").SetEnabled(false),
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

	dialer             *dialer.Lazy
	animation          *statusAnimation
	notificationID     string
	replaceCredentials atomic.Bool
	hasAccounts        atomic.Bool
	refreshingAccounts atomic.Bool
	state              atomic.Int32
	menu               struct {
		*appkit.MenuItem
		*appkit.Menu
		input struct {
			*appkit.InputMenuItem
			header *appkit.MenuItem
		}
		connect     *appkit.MenuItem
		cancel      *appkit.MenuItem
		disconnect  *appkit.MenuItem
		reset       *appkit.MenuItem
		editAccount *appkit.MenuItem
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
		Label:     l,
		URL:       u,
		updates:   make(chan dialer.StateUpdate),
		animation: pool.animation,
	}
	core.NewString(l).WithPointer(func(obj core.StringRef) {
		pd.menu.MenuItem = appkit.NewMenuItem().
			SetTag(int(actionDialer)).
			SetRepresentedObject(core.ObjectRef(obj))
		// Keep each action's identity fixed, including for queued clicks.
		pd.menu.connect = appkit.NewMenuItem().
			SetTitle("Connect").
			SetTag(int(actionConnect)).
			SetRepresentedObject(core.ObjectRef(obj)).
			SetCallback(pool.callback)
		pd.menu.cancel = appkit.NewMenuItem().
			SetTitle("Cancel").
			SetTag(int(actionCancel)).
			SetRepresentedObject(core.ObjectRef(obj)).
			SetCallback(pool.callback)
		pd.menu.disconnect = appkit.NewMenuItem().
			SetTitle("Disconnect").
			SetTag(int(actionDisconnect)).
			SetRepresentedObject(core.ObjectRef(obj)).
			SetCallback(pool.callback)
		pd.menu.reset = appkit.NewMenuItem().
			SetTitle("Reset").
			SetTag(int(actionReset)).
			SetRepresentedObject(core.ObjectRef(obj)).
			SetCallback(pool.callback)
		if usesKeychain(u) {
			pd.menu.editAccount = appkit.NewMenuItem().
				SetTitle("Edit Account…").
				SetHidden(true).
				SetTag(int(actionEditAccount)).
				SetRepresentedObject(core.ObjectRef(obj)).
				SetCallback(pool.callback)
		}
	})
	pd.menu.input.InputMenuItem = appkit.NewInputMenuItem("Enter authentication input")
	pd.menu.input.header = appkit.NewMenuItemSectionHeader("").
		SetHidden(true)
	pd.menu.input.SetHidden(true)
	pd.menu.Menu = appkit.NewMenu().SetAutoenablesItems(false).
		Append(pd.menu.input.header, pd.menu.input.MenuItem, pd.menu.connect, pd.menu.cancel, pd.menu.disconnect, pd.menu.reset)
	if pd.menu.editAccount != nil {
		pd.menu.Append(pd.menu.editAccount)
	}
	if usesKeychain(u) {
		target := weak.Make(&pd)
		pd.menu.Menu.SetWillOpen(func() {
			if pd := target.Value(); pd != nil {
				pd.refreshAccounts()
			}
		})
	}
	pd.menu.SetSubmenu(pd.menu.Menu)
	pd.updateMenu(dialer.Idle)
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
	state := dialer.Idle
	pd.refreshAccounts()
	defer func() {
		notify.Remove(pd.notificationID)
		pd.animation.updateState(state, dialer.Idle)
	}()
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
		pd.animation.updateState(state, update.State)
		pd.updateMenu(update.State)
		if update.State != dialer.Idle || state == dialer.Connected {
			pd.notification(update.State, update.Err)
		}
		state = update.State
	}
}

func (pd *PooledDialer) updateMenu(state dialer.ConnectionState) {
	pd.state.Store(int32(state))
	pd.menu.SetTitle(pd.Label).
		SetImage(pd.icon(state)).
		SetBadge(state.String())
	pd.menu.connect.SetHidden(state != dialer.Idle)
	pd.menu.cancel.SetHidden(state != dialer.Connecting)
	pd.menu.disconnect.SetHidden(state != dialer.Connected)
	pd.menu.reset.SetHidden(state != dialer.Failed)
	if pd.menu.editAccount != nil {
		pd.menu.editAccount.SetEnabled(state == dialer.Idle || state == dialer.Failed)
	}
}

func (pd *PooledDialer) Close() {
	pd.dialer.Close()
}

func (pd *PooledDialer) icon(state dialer.ConnectionState) *appkit.Image {
	var name string
	color := appkit.SymbolColorDefault
	switch state {
	case dialer.Idle:
		name = "circle"
	case dialer.Connected:
		name = "checkmark.circle.fill"
		color = appkit.SymbolColorGreen
	case dialer.Failed:
		name = "exclamationmark.circle.fill"
		color = appkit.SymbolColorRed
	case dialer.Connecting:
		name = "ellipsis.circle"
		color = appkit.SymbolColorOrange
	default:
		return nil
	}
	return appkit.ImageSystemSymbol(name, color)
}

func (pd *PooledDialer) notification(state dialer.ConnectionState, err error) {
	notify.Remove(pd.notificationID)
	pd.notificationID = fmt.Sprintf("status:%d", statusNotificationCounter.Add(1))
	notif := notify.Notification{
		Identifier:          pd.notificationID,
		Title:               pd.Label,
		PresentationOptions: notifications.PresentationOptionBanner,
	}
	switch state {
	case dialer.Idle:
		notif.Subtitle = "Proxy disconnected"
		notif.Body = "The connection was terminated."
	case dialer.Connecting:
		notif.Subtitle = "Connecting to proxy"
		notif.Body = "The connection to the proxy is being established."
	case dialer.Connected:
		notif.Subtitle = "Proxy connected"
		notif.Body = "The proxy connection has been established."
	case dialer.Failed:
		notif.Subtitle = "Proxy connection failed"
		notif.Body = ""
		if err != nil {
			notif.CategoryIdentifier = "proxy-failed"
			if pd.hasAccounts.Load() {
				notif.CategoryIdentifier = "proxy-auth-failed"
			}
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
	case "proxy-failed", "proxy-auth-failed":
		responses, cleanup := notify.NotifyCh(notif)
		go func() {
			defer cleanup()
			select {
			case response := <-responses:
				switch response.ActionIdentifier {
				case "proxy-reset":
					pd.dialer.Reset(dialer.Reset)
				case "proxy-edit-account":
					pd.editSavedAccount()
				}
			case <-pd.dialer.Done():
			}
		}()
	default:
		notify.Notify(notif)
	}
}

// Replacement is explicitly requested by the user. A failed attempt never
// mutates Keychain; Input commits the change after cookie acquisition.
func (pd *PooledDialer) editSavedAccount() {
	if !usesKeychain(pd.URL) {
		return
	}
	state := dialer.ConnectionState(pd.state.Load())
	if state != dialer.Idle && state != dialer.Failed {
		return
	}
	pd.replaceCredentials.Store(true)
	pd.dialer.Reset(dialer.Reset)
	pd.dialer.Reset(dialer.Connect)
}

// Inline passwords take precedence. Otherwise VPN credentials come from
// Keychain when requested by an authentication form, without a config switch.
func usesKeychain(u *url.URL) bool {
	password, _ := u.User.Password()
	return (u.Scheme == "anyconnect" || u.Scheme == "gp") && password == ""
}

// Refresh only account attributes, never password data. Called at startup,
// when this submenu opens, and after a successful credential save.
func (pd *PooledDialer) refreshAccounts() {
	if pd.menu.editAccount == nil || !pd.refreshingAccounts.CompareAndSwap(false, true) {
		return
	}
	defer pd.refreshingAccounts.Store(false)
	item, err := vpnAccount(pd.URL, pd.URL.User.Username())
	found := false
	if err == nil {
		accounts, err := item.Accounts()
		found = err == nil && len(accounts) != 0
	}
	pd.hasAccounts.Store(found)
	pd.menu.editAccount.SetHidden(!found)
}
