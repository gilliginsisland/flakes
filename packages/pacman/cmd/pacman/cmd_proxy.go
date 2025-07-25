package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/caseymrm/menuet"
	"github.com/gilliginsisland/pacman/pkg/dialer/ghost"
	"github.com/gilliginsisland/pacman/pkg/flagutil"
	"github.com/gilliginsisland/pacman/pkg/launch"
	"github.com/gilliginsisland/pacman/pkg/netutil"
	"github.com/gilliginsisland/pacman/pkg/proxy"
	"github.com/jessevdk/go-flags"
	"tailscale.com/net/socks5"
)

func init() {
	parser.AddCommand("proxy", "Run the proxy server", "Starts the proxy with specified options", &ProxyCommand{})
}

var _ flags.Commander = (*ProxyCommand)(nil)

type ProxyCommand struct {
	ListenAddr flagutil.HostPort `short:"l" long:"listen" default:"127.0.0.1:8080" description:"Listening address"`
	Launchd    bool              `long:"launchd" description:"Use launchd socket activation"`
	RulesFile  flagutil.File     `short:"f" long:"file" description:"Path to the rules file" required:"true"`
}

// Execute runs the proxy subcommand
func (c *ProxyCommand) Execute(args []string) error {
	app := menuet.App()
	app.Name = "PACman"
	app.Label = "com.github.gilliginsisland.pacman"
	app.NotificationResponder = func(id string, response string) {}

	app.HideStartup()
	app.SetMenuState(&menuet.MenuState{
		Image: "menuicon.pdf",
	})

	errCh := make(chan error, 1)
	go func() {
		err := c.run()
		errCh <- err
		os.Exit(1)
	}()

	app.RunApplication()

	return <-errCh
}

func (c *ProxyCommand) run() error {
	var rules ghost.RuleSet
	err := json.NewDecoder(&c.RulesFile).Decode(&rules)
	c.RulesFile.Close()
	if err != nil {
		return err
	}

	ghost := ghost.NewDialer(ghost.Opts{
		Matcher: ghost.CompileRuleSet(rules),
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
	})

	httpServer := proxy.NewServer(ghost, &proxy.PacHandler{Rules: rules})
	socksServer := socks5.Server{
		Dialer: ghost.DialContext,
		Logf: func(format string, v ...interface{}) {
			slog.Debug(fmt.Sprintf(format, v...))
		},
	}

	var listeners []net.Listener
	if c.Launchd {
		listeners, err = launch.ActivateSocket("Socket")
		if err != nil {
			return err
		}
		if len(listeners) == 0 {
			return errors.New("no launchd sockets were passed")
		}
	} else {
		l, err := net.Listen("tcp", string(c.ListenAddr))
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", c.ListenAddr, err)
		}
		listeners = []net.Listener{l}
	}

	var wg sync.WaitGroup
	for _, l := range listeners {
		slog.Info(
			"PACman proxy server starting",
			slog.String("address", l.Addr().String()),
		)

		mux := netutil.NewMuxListener(l)

		slog.Debug(
			"Mux listener created",
			slog.String("address", l.Addr().String()),
		)

		wg.Add(1)
		go func() {
			defer wg.Done()
			slog.Debug(
				"PACman http proxy server starting",
				slog.String("address", l.Addr().String()),
			)
			if err := http.Serve(mux.Http, httpServer); err != nil {
				slog.Error(fmt.Sprintf("http server stopped: %v\n", err))
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			slog.Debug(
				"PACman socks proxy server starting",
				slog.String("address", l.Addr().String()),
			)
			if err := socksServer.Serve(mux.Socks); err != nil {
				slog.Error(fmt.Sprintf("socks server stopped: %v\n", err))
			}
		}()
	}
	wg.Wait()

	slog.Info("PACman proxy server stopped")
	return nil
}
