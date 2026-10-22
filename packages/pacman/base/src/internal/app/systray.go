package app

import (
	"slices"

	"github.com/gilliginsisland/pacman/docs"
	"github.com/gilliginsisland/pacman/internal/app/plugin"
	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/menuet"
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
	pacman   *PACMan
	callback *core.Callback
	status   *appkit.StatusItem
	menu     *appkit.Menu
	dialers  []*appkit.MenuItem
}

func newAppKitMenu(pacman *PACMan, plugins []*plugin.Plugin) *systray {
	menu := &systray{
		pacman: pacman,
		status: appkit.NewStatusItem(),
		menu:   appkit.NewMenu(),
	}

	menu.callback = appkit.NewMenuItemCallback(menu.onAction)
	menu.dialers = pacman.pool.MenuItems()
	var pluginItems []*appkit.MenuItem
	for _, plugin := range plugins {
		pluginItems = append(pluginItems, plugin.MenuItems()...)
	}
	if len(pluginItems) > 0 {
		pluginItems = append([]*appkit.MenuItem{appkit.NewMenuItemSeparator()}, pluginItems...)
	}

	menu.status.SetTitle("PACman").
		SetImage(appkit.ImageNamed("menuicon.pdf")).
		SetMenu(menu.menu.
			Append(appkit.NewMenuItemSectionHeader("Server Address")).
			Append(appkit.NewMenuItem().
				SetTitle(menu.pacman.listener.Addr().String())).
			Append(appkit.NewMenuItemSeparator()).
			Append(appkit.NewMenuItemSectionHeader("Proxies")).
			Append(menu.dialers...).
			Append(appkit.NewMenuItemSeparator()).
			Append(appkit.NewMenuItem().
				SetTitle("Settings").
				SetSubmenu(appkit.NewMenu().
					Append(appkit.NewMenuItem().
						SetTitle("Edit").
						SetTag(int(actionEdit)).
						SetCallback(menu.callback)).
					Append(appkit.NewMenuItem().
						SetTitle("Reload").
						SetTag(int(actionReload)).
						SetCallback(menu.callback)).
					Append(appkit.NewMenuItemSeparator()).
					Append(appkit.NewMenuItem().
						SetTitle("Help").
						SetTag(int(actionHelp)).
						SetCallback(menu.callback)).
					Append(pluginItems...))).
			Append(appkit.NewMenuItem().
				SetTitle("Quit").
				SetTag(int(actionQuit)).
				SetCallback(menu.callback)))

	return menu
}

func (menu *systray) onAction(item *appkit.MenuItem) {
	action := appkitAction(item.Tag())
	switch action {
	case actionEdit:
		menu.pacman.OpenConfig()
	case actionReload:
		menu.pacman.ReloadConfig()
	case actionHelp:
		menuet.WebView(docs.HTML)
	case actionQuit:
		appkit.SharedApplication().Terminate()
	case actionDisconnect, actionReset, actionCancel, actionConnect:
		label := core.StringFromPointer(item.RepresentedObject()).String()
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
			menu.menu.Remove(item)
		}
	}
	for index, item := range items {
		if len(current) > 0 && current[0] == item {
			current = current[1:]
		} else {
			// Dialer rows follow the address section and Proxies header.
			menu.menu.Insert(item, 4+index)
		}
	}
	menu.dialers = items
}
