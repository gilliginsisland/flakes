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
	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/proxy"
)

// Config holds the configuration values
type Flags struct {
	ListenAddr flagutil.HostPort
	RulesFile  flagutil.File
}

func parseFlags(args []string) (*Flags, error) {
	f := Flags{
		ListenAddr: "127.0.0.1:8080",
	}

	flag.Var(&f.ListenAddr, "l", "Listening address (default 127.0.0.1:8080)")
	flag.Var(&f.RulesFile, "f", "Path to the rules file")

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
	slog.SetLogLoggerLevel(slog.LevelDebug)

	flags, err := parseFlags(os.Args[1:])
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	dialer := dialer.NewGHost(&net.Dialer{
		Timeout: 5 * time.Second,
	})
	if err = json.NewDecoder(flags.RulesFile).Decode(&dialer.Ruleset); err != nil {
		slog.Error(fmt.Sprintf("Error parsing rule file: %s", err))
		return
	}
	flags.RulesFile.Close()
	flags.RulesFile.File = nil

	server := proxy.NewProxyServer(dialer)

	slog.Info(fmt.Sprintf("PACman proxy server running on %s", string(flags.ListenAddr)))
	http.ListenAndServe(string(flags.ListenAddr), server)
}
