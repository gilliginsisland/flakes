package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/gilliginsisland/pacman/pkg/flagutil"
)

var opts struct {
	LogLevel flagutil.LogLevel `short:"v" long:"verbosity" description:"Verbosity level"`
}

// Global parser instance
var parser = flags.NewParser(&opts, flags.Default)

func init() {
	parser.CommandHandler = handleCommand
}

func handleCommand(cmd flags.Commander, args []string) error {
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: opts.LogLevel.Level,
		}),
	)
	slog.SetDefault(logger)

	slog.Debug(fmt.Sprintf("Running command: %#v", cmd))

	return cmd.Execute(args)
}
