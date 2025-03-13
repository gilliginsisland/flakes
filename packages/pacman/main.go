package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/gilliginsisland/pacman/pacman"
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

	dialer := pacman.Dialer{}
	if err := dialer.LoadRulesFile(rulesFile); err != nil {
		slog.Error(fmt.Sprintf("Error parsing rule file: %s", err))
		return
	}

	server := pacman.NewProxyServer(&dialer)

	slog.Info(fmt.Sprintf("PACman proxy server running on %s", *listenAddr))
	http.ListenAndServe(*listenAddr, server)
}
