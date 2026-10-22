package app

import (
	"errors"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/proxy"

	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/dialer/oc"
	"github.com/gilliginsisland/pacman/pkg/menuet"
	"github.com/gilliginsisland/pacman/pkg/menuet/plugin"
	"github.com/gilliginsisland/pacman/pkg/notify"
	"github.com/gilliginsisland/pacman/pkg/xdg"
)

type PACMan struct {
	config   Path
	pool     DialerPool
	dialer   dialer.ByHost
	listener net.Listener
	server   *ProxyServer
	menu     *systray
	mu       sync.Mutex
}

func Run(config Path, l net.Listener) error {
	var err error
	app := menuet.App()
	app.SetNotificationCategories(oc.NotificationCategories)
	app.Run(func() {
		defer app.Terminate()
		err = run(config, l)
		if err == nil {
			return
		}
		slog.Error("application terminated:", slog.Any("error", err))
		notify.Notify(notify.Notification{
			Subtitle: "Application Terminated",
			Body:     err.Error(),
		})
		app.Terminate()
	})
	return err
}

func run(config Path, l net.Listener) error {
	cfg, err := ParseConfigFile(config)
	if err != nil {
		return err
	}

	plugins, err := plugin.Load()
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
		pool: DialerPool{
			dialers: make(map[string]*PooledDialer),
		},
	}
	pacman.pool.fwd = &pacman.dialer
	pacman.menu = newAppKitMenu(&pacman, plugins)
	pacman.pool.callback = pacman.menu.callback
	pacman.server = NewProxyServer(&pacman.dialer)
	if err = pacman.LoadConfig(cfg); err != nil {
		return err
	}

	signalCh := make(chan os.Signal, 1)
	defer close(signalCh)
	signal.Notify(signalCh, syscall.SIGHUP)
	defer signal.Stop(signalCh)
	go func() {
		for range signalCh {
			pacman.ReloadConfig()
		}
	}()

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
	if err != nil {
		notify.Notify(notify.Notification{
			Title: "Config Reload Error",
			Body:  err.Error(),
		})
		return
	}
	notify.Notify(notify.Notification{
		Subtitle: "Config Reloaded",
		Body:     "The configuration was successfully reloaded.",
	})
}

func (pacman *PACMan) LoadConfig(cfg *Config) error {
	pacman.mu.Lock()
	defer pacman.mu.Unlock()

	// Validate against the incoming config before changing any live dialers.
	for _, r := range cfg.Rules {
		for _, proxy := range r.Proxies {
			if _, ok := cfg.Proxies[proxy]; !ok {
				return errors.New("proxy not found: " + proxy)
			}
		}
	}

	for k, u := range cfg.Proxies {
		pd := pacman.pool.dialers[k]
		if pd == nil || pd.URL.String() != u.String() {
			// Installed rules hold this exact Lazy. Keep it alive until the
			// replacement rules and menu are installed.
			if pd != nil {
				defer pd.Close()
			}
			pd = pacman.pool.NewPooledDialer(k, &u.URL)
			pacman.pool.dialers[k] = pd
		}
	}

	rs := dialer.RuleSet(BuildTrie(cfg, func(host string, proxies []string) proxy.ContextDialer {
		chain := make([]proxy.ContextDialer, len(proxies))
		for i, proxy := range proxies {
			pd := pacman.pool.dialers[proxy]
			chain[i] = pd.dialer
		}

		switch len(chain) {
		case 0:
			return nil
		case 1:
			suffix := strings.ToLower(proxies[0]) + ".pacman"
			if host == "."+suffix {
				return &dialer.RewritingDialer{
					Dialer: chain[0],
					Suffix: suffix,
				}
			}
			return chain[0]
		default:
			return dialer.Chain(chain)
		}
	}))

	pacman.dialer.Swap(&rs)

	for k, pd := range pacman.pool.dialers {
		if _, ok := cfg.Proxies[k]; ok {
			continue
		}
		delete(pacman.pool.dialers, k)
		defer pd.Close()
	}
	pacman.menu.updateDialers(&pacman.pool)

	return nil
}
