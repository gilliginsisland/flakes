package app

import (
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/caseymrm/menuet"
	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/netutil"
	"github.com/gilliginsisland/pacman/pkg/xdg"
)

type PACMan struct {
	config *Config
	pool   DialerPool
	dialer dialer.ByHost
	server netutil.Server
	menu   Menuer
	mu     sync.Mutex
}

func Run(configPath Path, l net.Listener) error {
	var err error
	go func() {
		err = run(configPath, l)
	}()
	menuet.App().RunApplication()
	return err
}

func run(configPath Path, l net.Listener) error {
	app := menuet.App()
	app.HideStartup()
	app.NotificationResponder = func(id string, response string) {}
	app.SetMenuState(&menuet.MenuState{
		Image: "menuicon.pdf",
	})

	cfg, err := ParseConfigFile(configPath)
	if err != nil {
		return err
	}

	pacman := PACMan{
		dialer: dialer.ByHost{
			Default: &net.Dialer{
				Timeout: 5 * time.Second,
			},
		},
		pool: make(DialerPool, len(cfg.Proxies)),
	}
	pacman.menu = Sections{
		Section{
			Title:   "Server Address",
			Content: &AddrItem{l},
		},
		Section{
			Title:   "Proxies",
			Content: pacman.pool,
		},
		StaticItem{
			Text:    "Edit RuleSet",
			Clicked: pacman.OpenConfig,
		},
	}
	app.Children = pacman.menu.MenuItems
	pacman.server = NewProxyServer(&pacman.dialer)

	if err = pacman.LoadConfig(cfg); err != nil {
		return err
	}

	slog.Info("PACman server listening", slog.String("address", l.Addr().String()))
	defer slog.Info("PACman proxy server stopped", slog.Any("error", err))
	return pacman.server.Serve(l)
}

func (pacman *PACMan) OpenConfig() {
	xdg.Run(pacman.config.Path.String())
}

func (pacman *PACMan) LoadConfig(cfg *Config) error {
	pacman.mu.Lock()
	defer pacman.mu.Unlock()

	pacman.config = cfg

	for k, u := range cfg.Proxies {
		// new PooledDialer if the URL has changed
		if pd := pacman.pool[k]; pd == nil || pd.URL.String() != u.String() {
			pacman.pool[k] = NewPooledDialer(k, &u.URL, &pacman.dialer)
		} else if pd != nil {
			// close existing dialer after we update the dialer ruleset
			defer pd.Close()
		}
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
