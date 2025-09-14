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
	config *Config
	dialer dialer.ByHost
	pool   map[string]*PooledDialer
	menuet *menuet.Application
	menu   *MainMenu
	server netutil.Server
}

var App = sync.OnceValue(func() *PACMan {
	app := menuet.App()
	app.Name = "PACman"
	app.Label = "io.github.gilliginsisland.pacman"
	app.NotificationResponder = func(id string, response string) {}
	app.SetMenuState(&menuet.MenuState{
		Image: "menuicon.pdf",
	})
	app.HideStartup()

	pacman := PACMan{
		menuet: app,
		dialer: dialer.ByHost{
			Default: &net.Dialer{
				Timeout: 5 * time.Second,
			},
		},
	}
	pacman.server = ProxyServer(&pacman.dialer)
	return &pacman
})

func (pacman *PACMan) LoadRuleSet(rs *Config) error {
	menu := RootMenu()
	menu.Settings.Clicked = func() {
		xdg.Run(rs.Path.String())
	}

	pool := make(map[string]*PooledDialer, len(rs.Proxies))
	for k, u := range rs.Proxies {
		var pd *PooledDialer
		pd, ok := pacman.pool[k]
		// create a new PooledDialer if the URL is new or has changed
		if !ok || pd.URL.String() != u.String() {
			pd = &PooledDialer{
				Label: k,
				URL:   &u.URL,
				Fwd:   &pacman.dialer,
			}
		}
		pool[k] = pd
	}

	for _, pd := range iterutil.SortedMapIter(pool) {
		pd.AttachMenu(menu.Proxies)
	}

	for k, pd := range pacman.pool {
		if _, ok := pool[k]; ok {
			continue
		}
		pd.lazy.Close()
	}

	byHost := &pacman.dialer
	for _, r := range rs.Rules {
		chain := make([]proxy.ContextDialer, len(r.Proxies))
		for i, proxy := range r.Proxies {
			pd, ok := pool[proxy]
			if !ok {
				return errors.New("proxy not found: " + proxy)
			}
			chain[i] = pd.Dialer()
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
			byHost.Add(h, pd)
		}
	}

	pacman.config = rs
	pacman.pool = pool
	pacman.menu = menu
	pacman.menuet.Children = menu.Children
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
