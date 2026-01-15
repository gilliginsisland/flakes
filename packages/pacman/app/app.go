package app

import (
	"errors"
	"log/slog"
	"net"
	"net/url"
	"os"
	"sync"
	"time"

	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/menuet"
	"github.com/gilliginsisland/pacman/pkg/netutil"
	"github.com/gilliginsisland/pacman/pkg/notify"
	"github.com/gilliginsisland/pacman/pkg/xdg"
)

type PACMan struct {
	config   Path
	pool     DialerPool
	dialer   dialer.ByHost
	listener net.Listener
	server   netutil.Server
	menu     Menuer
	mu       sync.Mutex
}

func Run(config Path, l net.Listener) error {
	var err error
	go func() {
		err = run(config, l)
		if err != nil {
			slog.Error("application terminated:", slog.Any("error", err))
			notify.Notify(notify.Notification{
				Title:   "Application Terminated",
				Message: err.Error(),
			})
			os.Exit(1)
		}
		os.Exit(0)
	}()
	menuet.App().RunApplication()
	return err
}

func run(config Path, l net.Listener) error {
	app := menuet.App()
	app.HideStartup()
	app.SetMenuState(&menuet.MenuState{
		Image: "menuicon.pdf",
	})

	cfg, err := ParseConfigFile(config)
	if err != nil {
		return err
	}

	if l == nil {
		if cfg.Listen == "" {
			return errors.New("no listener provided")
		} else {
			if l, err = net.Listen("tcp", cfg.Listen.String()); err != nil {
				return err
			}
		}
	}

	pacman := PACMan{
		config:   config,
		listener: l,
		dialer: dialer.ByHost{
			Default: &net.Dialer{
				Timeout: 5 * time.Second,
			},
		},
		pool: make(DialerPool),
	}
	pacman.menu = Sections{
		Section{
			Title: "Server Address",
			Content: AddrFuncerItem(func() net.Addr {
				return pacman.listener.Addr()
			}),
		},
		Section{
			Title:   "Proxies",
			Content: pacman.pool,
		},
		StaticItem{
			Text: "Settings",
			Children: (StaticItems{
				menuet.MenuItem{
					Text:    "Edit",
					Clicked: pacman.OpenConfig,
				},
				menuet.MenuItem{
					Text:    "Reload",
					Clicked: pacman.ReloadConfig,
				},
			}).MenuItems,
		},
	}
	pacman.server = NewProxyServer(&pacman.dialer)
	if err := pacman.LoadConfig(cfg); err != nil {
		return err
	}

	app.Children = pacman.menu.MenuItems
	slog.Info("PACman server listening", slog.String("address", l.Addr().String()))
	err = pacman.server.Serve(l)
	slog.Info("PACman proxy server stopped", slog.Any("error", err))
	return err
}

func (pacman *PACMan) OpenConfig() {
	p, err := pacman.config.ExpandUser()
	if err != nil {
		return
	}

	u := url.URL{
		Scheme: "file",
		Path:   p,
	}
	xdg.Run(u.String())
}

func (pacman *PACMan) ReloadConfig() {
	cfg, err := ParseConfigFile(pacman.config)
	if err == nil {
		err = pacman.LoadConfig(cfg)
	}
	if err == nil {
		return
	}
	notify.Notify(notify.Notification{
		Title:   "Config Reload Error",
		Message: err.Error(),
	})
}

func (pacman *PACMan) LoadConfig(cfg *Config) error {
	pacman.mu.Lock()
	defer pacman.mu.Unlock()

	for k, u := range cfg.Proxies {
		pd := pacman.pool[k]
		if pd != nil {
			// skip new PooledDialer if the URL has not changed
			if pd.URL.String() == u.String() {
				continue
			}
			// close existing dialer after we update the dialer ruleset
			defer pd.Close()
		}
		pacman.pool[k] = NewPooledDialer(k, &u.URL, &pacman.dialer)
	}

	var rs dialer.RuleSet
	for _, r := range cfg.Rules {
		chain := make([]proxy.ContextDialer, len(r.Proxies))
		for i, proxy := range r.Proxies {
			pd, ok := pacman.pool[proxy]
			if !ok {
				return errors.New("proxy not found: " + proxy)
			}
			chain[i] = pd.dialer
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
			rs.Add(h, pd)
		}
	}
	pacman.dialer.Swap(&rs)

	for k, pd := range pacman.pool {
		if _, ok := cfg.Proxies[k]; ok {
			continue
		}
		delete(pacman.pool, k)
		defer pd.Close()
	}
	menuet.App().MenuChanged()

	return nil
}
