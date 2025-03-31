package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gilliginsisland/pacman/internal/flagutil"
	"github.com/gilliginsisland/pacman/internal/syncutil"
	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/launch"
	"github.com/gilliginsisland/pacman/pkg/proxy"
)

func init() {
	parser.AddCommand("proxy", "Run the proxy server", "Starts the proxy with specified options", &ProxyCommand{})
}

type ProxyCommand struct {
	ListenAddr flagutil.HostPort `short:"l" long:"listen" default:"127.0.0.1:8080" description:"Listening address"`
	Launchd    bool              `long:"launchd" description:"Use launchd socket activation"`
}

// Execute runs the proxy subcommand
func (c *ProxyCommand) Execute(args []string) error {
	var rules dialer.Ruleset
	if err := json.NewDecoder(&opts.RulesFile).Decode(&rules); err != nil {
		return err
	}
	opts.RulesFile.Close()

	ghost := dialer.NewGHost(rules, &net.Dialer{
		Timeout: 5 * time.Second,
	})

	server := proxy.NewServer(ghost, &proxy.PacHandler{Rules: rules})

	if c.Launchd {
		listeners, err := launch.ActivateSocket("Socket")
		if err != nil || len(listeners) == 0 {
			return err
		}

		for l := range syncutil.ParallelRange(listeners) {
			slog.Info(
				"PACman proxy server starting",
				slog.String("address", l.Addr().String()),
			)
			if err := http.Serve(l, server); err != nil {
				slog.Error(fmt.Sprintf("server stopped: %v\n", err))
			}
		}
	} else {
		slog.Info(
			"PACman proxy server starting",
			slog.String("address", string(c.ListenAddr)),
		)
		http.ListenAndServe(string(c.ListenAddr), server)
	}

	slog.Info("PACman proxy server stopped")
	return nil
}
