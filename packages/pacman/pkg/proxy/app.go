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

func (a *Application) Serve(listeners []net.Listener) error {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	app := menuet.App()
	app.Name = "PACman"
	app.Label = "com.github.gilliginsisland.pacman"
	app.NotificationResponder = func(id string, response string) {}

	app.HideStartup()
	app.SetMenuState(&menuet.MenuState{
		Image: "menuicon.pdf",
	})

	app.Children = StaticMenu{{
		Text:     "Proxies",
		Children: a.proxiesMenu,
	}}.Children

	mux := netutil.ServeMux{}
	mux.Handle(netutil.SOCKS5Match, &socks5.Server{
		Dialer: a.dialer.DialContext,
		Logf: func(format string, v ...any) {
			slog.Debug(fmt.Sprintf(format, v...))
		},
	})
	mux.Handle(netutil.DefaultMatch, &Server{
		Dialer: a.dialer.DialContext,
		Handler: &PacHandler[proxy.ContextDialer]{
			Trie: a.trie,
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

func (a *Application) proxiesMenu() []menuet.MenuItem {
	s := make([]menuet.MenuItem, 0, len(a.pool))
	for n, d := range a.pool {
		connected := d.ContextDialer != nil

		var icon string
		if connected {
			icon = "🟢 "
		} else {
			icon = "⚪ "
		}

		s = append(s, menuet.MenuItem{
			Text: icon + n,
			Children: func() []menuet.MenuItem {
				var action string
				if connected {
					action = "Disconnect"
				} else {
					action = "Connect"
				}

				return []menuet.MenuItem{
					{
						Text: action,
						Clicked: func() {
							if connected {
								d.Close()
							}
						},
					},
				}
			},
		})
	}
	return s
}
