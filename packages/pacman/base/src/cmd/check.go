package cmd

import (
	"os"
	"strings"

	"github.com/jessevdk/go-flags"

	"github.com/gilliginsisland/pacman/internal/app"
)

func init() {
	parser.AddCommand("check", "Check host rules", "Check if a host matches the ruleset", &CheckCmd{})
}

var _ flags.Commander = (*CheckCmd)(nil)

// CheckCmd defines the "check" command.
type CheckCmd struct{}

// Execute runs the check command.
func (c *CheckCmd) Execute(args []string) error {
	rs, err := app.ParseConfigFile(opts.ConfigPath)
	if err != nil {
		return err
	}

	t := app.BuildTrie(rs, func(string, []string) struct{} { return struct{}{} })

	for _, host := range args {
		if _, ok := t.Match(strings.ToLower(host)); ok {
			return nil
		}
	}

	os.Exit(1)

	return nil
}
