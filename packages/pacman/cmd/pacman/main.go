package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/proxy"
)

func main() {
	listenAddr := flag.String("l", "127.0.0.1:8080", "Listening address (default 127.0.0.1:8080)")
	rulesFilename := flag.String("f", "", "Path to the rules file")
	flag.Parse()

	if _, _, err := net.SplitHostPort(*listenAddr); err != nil {
		slog.Error(fmt.Sprintf("Error: invalid listening address format: %s\n", err))
		flag.Usage()
		os.Exit(1)
	}

	if *rulesFilename == "" {
		slog.Error("rules file is required")
		flag.Usage()
		os.Exit(1)
	}

	rulesFile, err := os.Open(*rulesFilename)
	if err != nil {
		slog.Error(fmt.Sprintf("Error: could not open rules file at path: %s, %s\n", *rulesFilename, err))
		os.Exit(1)
	}
	defer rulesFile.Close()

	dialer := dialer.NewGHost()
	if err = json.NewDecoder(rulesFile).Decode(&dialer.Ruleset); err != nil {
		slog.Error(fmt.Sprintf("Error parsing rule file: %s", err))
		return
	}

	server := proxy.NewProxyServer(dialer)

	slog.Info(fmt.Sprintf("PACman proxy server running on %s", *listenAddr))
	http.ListenAndServe(*listenAddr, server)
}
