package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gilliginsisland/pacman/internal/flagutil"
	"github.com/gilliginsisland/pacman/internal/syncutil"
	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/launch"
	"github.com/gilliginsisland/pacman/pkg/proxy"
)

// Config holds the configuration values
type Flags struct {
	ListenAddr flagutil.HostPort
	RulesFile  flagutil.File
	LogLevel   flagutil.LogLevel
	Launchd    bool
}

func parseFlags(args []string) (*Flags, error) {
	f := Flags{
		ListenAddr: "127.0.0.1:8080",
		LogLevel: flagutil.LogLevel{
			Level: slog.LevelInfo,
		},
	}

	flag.Var(&f.ListenAddr, "l", "Listening address (default 127.0.0.1:8080)")
	flag.Var(&f.ListenAddr, "listen", "Listening address (default 127.0.0.1:8080)")

	flag.Var(&f.RulesFile, "f", "Path to the rules file")
	flag.Var(&f.RulesFile, "file", "Path to the rules file")

	flag.Var(&f.LogLevel, "v", "Verbosity")
	flag.Var(&f.LogLevel, "verbosity", "Verbosity")

	flag.BoolVar(&f.Launchd, "launchd", false, "Use launchd socket activation")

	// Manually parse arguments
	if err := flag.CommandLine.Parse(args); err != nil {
		return nil, err
	}

	if f.RulesFile.File == nil {
		return nil, errors.New("rules file is required")
	}

	return &f, nil
}

func main() {
	// logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	// slog.SetDefault(logger)

	flags, err := parseFlags(os.Args[1:])
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	slog.SetLogLoggerLevel(flags.LogLevel.Level)

	var rules dialer.Ruleset
	if err = json.NewDecoder(flags.RulesFile).Decode(&rules); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	flags.RulesFile.Close()
	flags.RulesFile.File = nil

	ghost := dialer.NewGHost(rules, &net.Dialer{
		Timeout: 5 * time.Second,
	})

	server := proxy.NewServer(ghost, &proxy.PacHandler{Rules: rules})

	if flags.Launchd {
		listeners, err := launch.ActivateSocket("Socket")
		if err != nil || len(listeners) == 0 {
			slog.Error(err.Error())
			os.Exit(1)
		}

		for l := range syncutil.ParallelRange(listeners) {
			slog.Info("PACman proxy server starting", slog.String("address", l.Addr().String()))
			if err := http.Serve(l, server); err != nil {
				slog.Error(fmt.Sprintf("server stopped: %v\n", err))
			}
		}
	} else {
		slog.Info("PACman proxy server starting", slog.String("address", flags.ListenAddr.String()))
		http.ListenAndServe(string(flags.ListenAddr), server)
	}

	slog.Info("PACman proxy server stopped")
}
