package app

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
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
	pool   map[string]*dialer.Lazy
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

	var menu Menu
	app.Children = menu.Children

	pacman := PACMan{
		menuet: app,
		menu:   RootMenu(&menu),
	}
	pacman.server = ProxyServer(&pacman.dialer)
	return &pacman
})

func (pacman *PACMan) LoadRuleSet(rs *RuleSet) error {
	pacman.menu.Settings.Clicked = func() {
		xdg.Run(pacman.rs.Path)
	}

	pool := make(map[string]*dialer.Lazy, len(rs.Proxies))
	byHost := dialer.ByHost{
		Default: &net.Dialer{
			Timeout: 5 * time.Second,
		},
	}

	for k, u := range iterutil.SortedMapIter(rs.Proxies) {
		menu := DialerMenuItem{
			label: k,
			node:  pacman.menu.Proxies.AddChild(menuet.MenuItem{}),
		}
		menu.child = menu.node.AddChild(menuet.MenuItem{})

		var timeout time.Duration = 1 * time.Hour
		if t := u.Query().Get("timeout"); t != "" {
			if i, err := strconv.Atoi(t); err == nil {
				timeout = time.Duration(i) * time.Second
			}
		}
		menu.lazy = &dialer.Lazy{
			Timeout: timeout,
			New: func() (proxy.ContextDialer, error) {
				return FromURL(&u.URL, &byHost, menu.StateChanged)
			},
		}
		menu.StateChanged(dialer.Offline)

		pool[k] = menu.lazy
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
			byHost.Add(h, pd)
		}
	}

	pacman.rs = rs
	pacman.pool = pool
	pacman.dialer = byHost
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
