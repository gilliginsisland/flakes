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
	"github.com/gilliginsisland/pacman/pkg/netutil"
	"github.com/gilliginsisland/pacman/pkg/trie"
	"github.com/gilliginsisland/pacman/pkg/xdg"
)

type Application struct {
	*menuet.Application
	rs     *RuleSet
	dialer dialer.ByHost
	trie   *trie.Host[proxy.ContextDialer]
	pool   map[string]*dialer.Lazy
}

func App(rs *RuleSet) (*Application, error) {
	pool := make(map[string]*dialer.Lazy, len(rs.Proxies))
	trie := trie.NewHost[proxy.ContextDialer]()
	nd := net.Dialer{
		Timeout: 5 * time.Second,
	}

	byHost := dialer.ByHost(func(host string) proxy.ContextDialer {
		pd, found := trie.Match(host)
		if !found {
			pd = &nd
		}
		return pd
	})

	for k, u := range rs.Proxies {
		var timeout time.Duration = 1 * time.Hour
		if t := u.Query().Get("timeout"); t != "" {
			if i, err := strconv.Atoi(t); err == nil {
				timeout = time.Duration(i) * time.Second
			}
		}
		init := func() (proxy.ContextDialer, error) {
			slog.Debug(
				"Creating dialer",
				slog.String("proxy", u.Redacted()),
			)
			return FromURL(&u.URL, byHost)
		}
		pd := dialer.NewLazy(init, timeout)
		pool[k] = pd
	}

	for _, r := range rs.Rules {
		chain := make([]proxy.ContextDialer, len(r.Proxies))
		for i, proxy := range r.Proxies {
			pd, ok := pool[proxy]
			if !ok {
				return nil, errors.New("proxy not found: " + proxy)
			}
			chain[i] = pd
		}

		var pd proxy.ContextDialer
		switch len(chain) {
		case 0:
			pd = &nd
		case 1:
			pd = chain[0]
		default:
			pd = dialer.Chain(chain)
		}
		for _, h := range r.Hosts {
			trie.Insert(h, pd)
		}
	}

	return &Application{
		rs:          rs,
		trie:        trie,
		pool:        pool,
		dialer:      byHost,
		Application: menuet.App(),
	}, nil
}

func (app *Application) Serve(listeners []net.Listener) error {
	app.Name = "PACman"
	app.Label = "com.github.gilliginsisland.pacman"
	app.NotificationResponder = func(id string, response string) {}
	app.SetMenuState(&menuet.MenuState{
		Image: "menuicon.pdf",
	})

	menu := MenuNode{}
	app.Children = menu.Children
	menu.AddChild(menuet.MenuItem{
		Text:       "Server Address",
		FontWeight: menuet.WeightMedium,
	})
	for _, l := range listeners {
		menu.AddChild(menuet.MenuItem{
			Text:       l.Addr().String(),
			FontWeight: menuet.WeightLight,
		})
	}
	menu.AddChild(menuet.MenuItem{
		Type: menuet.Separator,
	})
	menu.AddChild(menuet.MenuItem{
		Text:       "Proxies",
		FontWeight: menuet.WeightMedium,
	})
	buildProxiesMenu(&menu, app.pool)
	menu.AddChild(menuet.MenuItem{
		Type: menuet.Separator,
	})
	settings := menu.AddChild(menuet.MenuItem{
		Text: "Settings",
	})
	settings.AddChild(menuet.MenuItem{
		Text: "Edit",
		Clicked: func() {
			xdg.Run(app.rs.Path)
		},
	})

	mux := netutil.ServeMux{}
	mux.Handle(netutil.SOCKS5Match, &socks5.Server{
		Dialer: app.dialer.DialContext,
		Logf: func(format string, v ...any) {
			slog.Debug(fmt.Sprintf(format, v...))
		},
	})
	mux.Handle(netutil.DefaultMatch, &httpproxy.Server{
		Dialer: app.dialer.DialContext,
		Handler: &httpproxy.PacHandler[proxy.ContextDialer]{
			Trie: app.trie,
		},
	})

	var wg sync.WaitGroup
	for _, l := range listeners {
		wg.Add(1)
		go func() {
			defer wg.Done()
			slog.Info(
				"PACman server listening",
				slog.String("address", l.Addr().String()),
			)
			mux.Serve(l)
		}()
	}
	app.RunApplication()
	wg.Wait()

	return nil
}

func buildProxiesMenu(menu *MenuNode, pool map[string]*dialer.Lazy) {
	for l, d := range pool {
		node := menu.AddChild(menuet.MenuItem{})
		child := node.AddChild(menuet.MenuItem{})
		refresh := func(state dialer.ConnectionState) {
			node.Text = icon(state) + " " + l
			child.Text, child.Clicked = action(state, d)
		}
		refresh(d.State())

		ch := d.Observe()
		go func() {
			for state := range ch {
				refresh(state())
			}
		}()
	}
}

func icon(state dialer.ConnectionState) string {
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

func action(state dialer.ConnectionState, d *dialer.Lazy) (string, func()) {
	switch state {
	case dialer.Offline, dialer.Failed:
		return "Connect", nil
	case dialer.Online:
		return "Disconnect", func() { d.Close() }
	case dialer.Connecting:
		return "🟡", nil
	}
	return "", nil
}
