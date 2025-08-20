package proxy

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/caseymrm/menuet"
	"golang.org/x/net/proxy"
	"tailscale.com/net/socks5"

	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/netutil"
	"github.com/gilliginsisland/pacman/pkg/trie"
)

type Application struct {
	*menuet.Application
	rs     RuleSet
	dialer dialer.ByHost
	trie   *trie.Host[proxy.ContextDialer]
	pool   map[string]*dialer.Lazy
}

func App(rs RuleSet) (*Application, error) {
	nd := net.Dialer{
		Timeout: 5 * time.Second,
	}

	var app *Application
	app = &Application{
		rs:   rs,
		trie: trie.NewHost[proxy.ContextDialer](),
		pool: make(map[string]*dialer.Lazy, len(rs.Proxies)),
		dialer: dialer.ByHost(func(host string) proxy.ContextDialer {
			if pd, found := app.trie.Match(host); found {
				return pd
			} else {
				return &nd
			}
		}),
		Application: menuet.App(),
	}

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
			return FromURL(&u.URL, app.dialer)
		}
		pd := dialer.NewLazy(init, timeout)
		app.pool[k] = pd
	}

	for _, r := range rs.Rules {
		chain := make([]proxy.ContextDialer, len(r.Proxies))
		for i, proxy := range r.Proxies {
			pd, ok := app.pool[proxy]
			if !ok {
				return nil, errors.New("proxy not found: " + proxy)
			}
			chain[i] = pd
		}

		var pd proxy.ContextDialer
		switch len(chain) {
		case 0:
			continue
		case 1:
			pd = chain[0]
		default:
			pd = dialer.Chain(chain)
		}
		for _, h := range r.Hosts {
			app.trie.Insert(h, pd)
		}
	}

	return app, nil
}

func (app *Application) Serve(listeners []net.Listener) error {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	app.Name = "PACman"
	app.Label = "com.github.gilliginsisland.pacman"
	app.NotificationResponder = func(id string, response string) {}

	app.HideStartup()
	app.SetMenuState(&menuet.MenuState{
		Image: "menuicon.pdf",
	})

	menu := MenuNode{}
	app.Children = menu.Children
	buildProxiesMenu(menu.AddChild(menuet.MenuItem{
		Text: "Proxies",
	}), app.pool)

	mux := netutil.ServeMux{}
	mux.Handle(netutil.SOCKS5Match, &socks5.Server{
		Dialer: app.dialer.DialContext,
		Logf: func(format string, v ...any) {
			slog.Debug(fmt.Sprintf(format, v...))
		},
	})
	mux.Handle(netutil.DefaultMatch, &Server{
		Dialer: app.dialer.DialContext,
		Handler: &PacHandler[proxy.ContextDialer]{
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
