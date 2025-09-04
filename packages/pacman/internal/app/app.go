package app

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/caseymrm/menuet"
	"golang.org/x/net/proxy"
	"tailscale.com/net/socks5"

	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/httpproxy"
	"github.com/gilliginsisland/pacman/pkg/iterutil"
	"github.com/gilliginsisland/pacman/pkg/netutil"
	"github.com/gilliginsisland/pacman/pkg/xdg"
)

type PACMan struct {
	rs     *RuleSet
	dialer dialer.ByHost
	menuet *menuet.Application
	menu   *MainMenu
	server netutil.Server
}

var App = sync.OnceValue(func() *PACMan {
	app := menuet.App()
	app.Name = "PACman"
	app.Label = "com.github.gilliginsisland.pacman"
	app.NotificationResponder = func(id string, response string) {}
	app.SetMenuState(&menuet.MenuState{
		Image: "menuicon.pdf",
	})

	menu := RootMenu()
	app.Children = menu.Children

	pacman := PACMan{
		menuet: app,
		menu:   menu,
		dialer: dialer.ByHost{
			Default: &net.Dialer{
				Timeout: 5 * time.Second,
			},
		},
	}
	pacman.server = ProxyServer(&pacman.dialer)
	return &pacman
})

func (pacman *PACMan) LoadRuleSet(rs *RuleSet) error {
	pacman.menu.Settings.Clicked = func() {
		xdg.Run(pacman.rs.Path)
	}

	pool := make(map[string]proxy.ContextDialer, len(rs.Proxies))
	for k, u := range iterutil.SortedMapIter(rs.Proxies) {
		pd := PooledDialer{
			Label: k,
			URL:   &u.URL,
			Fwd:   &pacman.dialer,
		}
		pd.AttachMenu(pacman.menu.Proxies)
		pool[k] = pd.Dialer()
	}

	for _, r := range rs.Rules {
		chain := make([]proxy.ContextDialer, len(r.Proxies))
		for i, proxy := range r.Proxies {
			pd, ok := pool[proxy]
			if !ok {
				return errors.New("proxy not found: " + proxy)
			}
			chain[i] = pd
		}

		var pd proxy.ContextDialer
		switch len(chain) {
		case 0:
			pd = nil
		case 1:
			pd = chain[0]
		default:
			pd = dialer.Chain(chain)
		}

		for _, h := range r.Hosts {
			pacman.dialer.Add(h, pd)
		}
	}

	pacman.rs = rs
	return nil
}

func (pacman *PACMan) Serve(l net.Listener) error {
	pacman.menu.Server.AddChild(menuet.MenuItem{
		Text:       l.Addr().String(),
		FontWeight: menuet.WeightLight,
	})
	return pacman.server.Serve(l)
}

func (pacman *PACMan) RunApplication() {
	pacman.menuet.RunApplication()
}

func ProxyServer(pd *dialer.ByHost) netutil.Server {
	var mux netutil.ServeMux
	mux.Handle(netutil.SOCKS5Match, &socks5.Server{
		Dialer: pd.DialContext,
		Logf: func(format string, v ...any) {
			slog.Debug(fmt.Sprintf(format, v...))
		},
	})
	mux.Handle(netutil.DefaultMatch, &httpproxy.Server{
		Dialer:  pd.DialContext,
		Handler: &httpproxy.PacHandler{Hosts: pd.Hosts},
	})
	return &mux
}

type MainMenu struct {
	Menu
	Server   *MenuGroup
	Proxies  *MenuGroup
	Settings *MenuNode
}

func RootMenu() *MainMenu {
	var m MainMenu

	m.Server = m.Menu.AddGroup()
	m.Server.AddChild(menuet.MenuItem{
		Text:       "Server Address",
		FontWeight: menuet.WeightMedium,
	})

	m.Proxies = m.Menu.AddGroup()
	m.Proxies.AddChild(menuet.MenuItem{
		Text:       "Proxies",
		FontWeight: menuet.WeightMedium,
	})

	m.Settings = m.Menu.AddGroup().AddChild(
		menuet.MenuItem{
			Text: "Edit RuleSet",
		},
	)

	return &m
}
