package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"

	_ "net/http/pprof"

	"github.com/jessevdk/go-flags"

	"github.com/gilliginsisland/pacman/pkg/flagutil"
	"github.com/gilliginsisland/pacman/pkg/launch"
	"github.com/gilliginsisland/pacman/pkg/proxy"
)

func init() {
	parser.AddCommand("proxy", "Run the proxy server", "Starts the proxy with specified options", &ProxyCommand{})
}

var _ flags.Commander = (*ProxyCommand)(nil)

type ProxyCommand struct {
	ListenAddr flagutil.HostPort `short:"l" long:"listen" default:"127.0.0.1:8080" description:"Listening address"`
	Launchd    bool              `long:"launchd" description:"Use launchd socket activation"`
}

// Execute runs the proxy subcommand
func (c *ProxyCommand) Execute(args []string) error {
	var rules proxy.RuleSet
	err := json.NewDecoder(&opts.RulesFile).Decode(&rules)
	opts.RulesFile.Close()
	if err != nil {
		return err
	}

	app, err := proxy.App(rules)
	if err != nil {
		return err
	}

	listeners, err := c.listeners()
	if err != nil {
		return err
	}

	err = app.Serve(listeners)
	slog.Info("PACman proxy server stopped")
	return err
}

func (c *ProxyCommand) listeners() (listeners []net.Listener, err error) {
	if c.Launchd {
		if listeners, err = launch.ActivateSocket("Socket"); err != nil {
			return nil, err
		}
		if len(listeners) == 0 {
			return nil, errors.New("no launchd sockets were passed")
		}
	} else {
		if l, err := net.Listen("tcp", string(c.ListenAddr)); err != nil {
			return nil, fmt.Errorf("failed to listen on %s: %w", c.ListenAddr, err)
		} else {
			listeners = []net.Listener{l}
		}
	}
	return listeners, nil
}
