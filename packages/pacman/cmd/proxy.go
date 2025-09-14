package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"net"

	_ "net/http/pprof"

	"github.com/jessevdk/go-flags"

	"github.com/gilliginsisland/pacman/app"
	"github.com/gilliginsisland/pacman/pkg/flagutil"
	"github.com/gilliginsisland/pacman/pkg/launch"
)

func init() {
	parser.AddCommand("proxy", "Run the proxy server", "Starts the proxy with specified options", &ProxyCommand{})
}

var _ flags.Commander = (*ProxyCommand)(nil)

type ProxyCommand struct {
	ListenAddr flagutil.HostPort `short:"l" long:"listen" default:"127.0.0.1:11078" description:"Listening address"`
	Launchd    bool              `long:"launchd" description:"Use launchd socket activation"`
}

// Execute runs the proxy subcommand
func (c *ProxyCommand) Execute(args []string) error {
	rules, err := app.ParseConfigFile(opts.ConfigPath)
	if err != nil {
		return err
	}

	app := app.App()
	if err = app.LoadRuleSet(rules); err != nil {
		return err
	}

	l, err := c.listener()
	if err != nil {
		return err
	}

	go func() {
		slog.Info(
			"PACman server listening",
			slog.String("address", l.Addr().String()),
		)
		err = app.Serve(l)
		slog.Info("PACman proxy server stopped")
	}()
	app.RunApplication()

	return err
}

func (c *ProxyCommand) listener() (net.Listener, error) {
	if c.Launchd {
		listeners, err := launch.ActivateSocket("Socket")
		if err != nil {
			return nil, err
		}
		if len(listeners) == 0 {
			return nil, errors.New("no launchd sockets were passed")
		}
		return listeners[0], nil
	}

	l, err := net.Listen("tcp", string(c.ListenAddr))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", c.ListenAddr, err)
	}

	return l, nil
}
