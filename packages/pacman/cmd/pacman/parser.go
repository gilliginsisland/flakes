package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jessevdk/go-flags"
)

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
