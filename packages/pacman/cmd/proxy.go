package cmd

import (
	"errors"
	"net"

	_ "net/http/pprof"

	"github.com/jessevdk/go-flags"

	"github.com/gilliginsisland/pacman/app"
	"github.com/gilliginsisland/pacman/pkg/launch"
)

func init() {
	parser.AddCommand("proxy", "Run the proxy server", "Starts the proxy with specified options", &ProxyCommand{})
}

var _ flags.Commander = (*ProxyCommand)(nil)

type ProxyCommand struct {
	Launchd bool `long:"launchd" description:"Use launchd socket activation"`
}

// Execute runs the proxy subcommand
func (c *ProxyCommand) Execute(args []string) error {
	var l net.Listener
	if c.Launchd {
		var err error
		if l, err = c.listener(); err != nil {
			return err
		}
	}
	return app.Run(opts.ConfigPath, l)
}

func (c *ProxyCommand) listener() (net.Listener, error) {
	listeners, err := launch.ActivateSocket("Socket")
	if err != nil {
		return nil, err
	}
	if len(listeners) == 0 {
		return nil, errors.New("no launchd sockets were passed")
	}
	return listeners[0], nil
}
