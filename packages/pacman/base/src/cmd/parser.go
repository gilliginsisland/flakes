package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/gilliginsisland/pacman/internal/app"
	"github.com/gilliginsisland/pacman/pkg/flagutil"
)

var opts Opts

type Opts struct {
	ConfigPath app.Path          `short:"c" long:"config" description:"Path to the config file"`
	LogLevel   flagutil.LogLevel `short:"v" long:"verbosity" description:"Verbosity level"`
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

	if opts.ConfigPath == "" {
		path, err := app.EnsureDefaultConfigFile()
		if err != nil {
			return err
		}
		opts.ConfigPath = path
	}

	return cmd.Execute(args)
}

func ParseArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		args = []string{"proxy"}
	} else if args[0] == "" {
		// for csd wrapper, there is no way to pass extra args
		// also the flags are sent in an incompatible way
		if _, ok := os.LookupEnv("CSD_TOKEN"); ok {
			args = []string{"csd"}
		}
	}

	return parser.ParseArgs(args)
}
