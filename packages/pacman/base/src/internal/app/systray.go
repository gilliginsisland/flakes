package app

import (
	"slices"

	"github.com/gilliginsisland/pacman/internal/app/plugin"
	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/menuet/appkit"
	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type appkitAction int

const (
	actionEdit appkitAction = iota + 1
	actionReload
	actionHelp
	actionQuit
	actionDialer
	// Dialer actions follow the static tags and preserve their ResetAction values.
	actionCancel     = actionDialer + appkitAction(dialer.Cancel)
	actionDisconnect = actionDialer + appkitAction(dialer.Disconnect)
	actionReset      = actionDialer + appkitAction(dialer.Reset)
	actionConnect    = actionDialer + appkitAction(dialer.Connect)
)

type systray struct {
	*appkit.StatusItem
	*appkit.Menu

	pacman      *PACMan
	callback    *core.Callback
	dialers     []*appkit.MenuItem
	proxySwitch *appkit.SwitchMenuItem
}

func newAppKitMenu(pacman *PACMan, plugins []*plugin.Plugin) *systray {
	tray := &systray{
		StatusItem:  appkit.NewStatusItem(),
		Menu:        appkit.NewMenu().SetAutoenablesItems(false),
		pacman:      pacman,
		proxySwitch: appkit.NewSwitchMenuItem("PACman"),
	}
	tray.callback = appkit.NewMenuItemCallback(tray.onAction)
	tray.proxySwitch.Switch.SetEnabled(false)
	tray.proxySwitch.Switch.SetSpinning(true)
	tray.Append(tray.proxySwitch.MenuItem)
	tray.dialers = pacman.pool.MenuItems()
	var pluginItems []*appkit.MenuItem
	for _, plugin := range plugins {
		pluginItems = append(pluginItems, plugin.MenuItems()...)
	}
	if len(pluginItems) > 0 {
		pluginItems = append([]*appkit.MenuItem{appkit.NewMenuItemSeparator()}, pluginItems...)
	}

	tray.Append(appkit.NewMenuItemSeparator()).
		Append(appkit.NewMenuItemSectionHeader("Proxies").SetTitleFont(13, appkit.FontWeightSemibold)).
		Append(tray.dialers...).
		Append(appkit.NewMenuItemSeparator()).
		Append(appkit.NewMenuItem().
			SetTitle("Settings").
			SetSubmenu(appkit.NewMenu().
				Append(appkit.NewMenuItemSectionHeader("Server Address").SetTitleFont(13, appkit.FontWeightSemibold)).
				Append(appkit.NewMenuItem().
					SetTitle(tray.pacman.listener.Addr().String())).
				Append(appkit.NewMenuItemSeparator()).
				Append(appkit.NewMenuItem().
					SetTitle("Edit").
					SetTag(int(actionEdit)).
					SetCallback(tray.callback)).
				Append(appkit.NewMenuItem().
					SetTitle("Reload").
					SetTag(int(actionReload)).
					SetCallback(tray.callback)).
				Append(appkit.NewMenuItemSeparator()).
				Append(appkit.NewMenuItem().
					SetTitle("Help").
					SetTag(int(actionHelp)).
					SetCallback(tray.callback)).
				Append(pluginItems...))).
		Append(appkit.NewMenuItemSeparator()).
		Append(appkit.NewMenuItem().
			SetTitle("Quit PACman").
			SetTag(int(actionQuit)).
			SetCallback(tray.callback))

	tray.SetTitle("PACman").
		SetImage(appkit.ImageNamed("menuicon.pdf")).
		SetMenu(tray.Menu)

	return tray
}

func (menu *systray) onAction(item *appkit.MenuItem) {
	action := appkitAction(item.Tag())
	switch action {
	case actionEdit:
		menu.pacman.OpenConfig()
	case actionReload:
		menu.pacman.ReloadConfig()
	case actionHelp:
		appkit.SharedApplication().ShowHelp()
	case actionQuit:
		appkit.SharedApplication().Terminate()
	case actionDisconnect, actionReset, actionCancel, actionConnect:
		label := core.StringFromPointer(core.StringRef(item.RepresentedObject())).String()
		menu.pacman.mu.Lock()
		pd := menu.pacman.pool.dialers[label]
		menu.pacman.mu.Unlock()
		if pd == nil {
			return
		}
		pd.dialer.Reset(dialer.ResetAction(action - actionDialer))
	}
}

func (menu *systray) updateDialers(pool *DialerPool) {
	items := pool.MenuItems()
	current := menu.dialers[:0]
	for _, item := range menu.dialers {
		if slices.Contains(items, item) {
			current = append(current, item)
		} else {
			menu.Menu.Remove(item)
		}
	}
	for index, item := range items {
		if len(current) > 0 && current[0] == item {
			current = current[1:]
		} else {
			// Dialer rows follow the PACman switch, separator, and Proxies header.
			menu.Menu.Insert(item, 3+index)
		}
	}
	menu.dialers = items
}
