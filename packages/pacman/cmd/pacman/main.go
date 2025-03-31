package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gilliginsisland/pacman/internal/flagutil"
	"github.com/jessevdk/go-flags"
)

var opts struct {
	RulesFile flagutil.File     `short:"f" long:"file" description:"Path to the rules file" required:"true"`
	LogLevel  flagutil.LogLevel `short:"v" long:"verbosity" description:"Verbosity level"`
}

// Global parser instance
var parser = flags.NewParser(&opts, flags.Default)

func init() {
	parser.CommandHandler = func(cmd flags.Commander, args []string) error {
		logger := slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: opts.LogLevel.Level,
			}),
		)
		slog.SetDefault(logger)

		slog.Info(fmt.Sprintf("Setting Log Level %s", opts.LogLevel.Level.String()))

		return cmd.Execute(args)
	}
}

func main() {
	// Parse and execute subcommands
	_, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}
}
