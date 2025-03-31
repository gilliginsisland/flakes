package main

import (
	"log/slog"
	"os"

	"github.com/gilliginsisland/pacman/internal/flagutil"
	"github.com/jessevdk/go-flags"
)

var opts struct {
	RulesFile flagutil.File `short:"f" long:"file" description:"Path to the rules file" required:"true"`
	LogLevel  slog.Level    `short:"v" long:"verbosity" description:"Verbosity level"`
}

// Global parser instance
var parser = flags.NewParser(&opts, flags.Default)

func init() {
	parser.CommandHandler = func(cmd flags.Commander, args []string) error {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
		slog.SetLogLoggerLevel(opts.LogLevel)

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
