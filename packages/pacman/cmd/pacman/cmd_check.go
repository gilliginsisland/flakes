package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gilliginsisland/pacman/pkg/dialer/ghost"
	"github.com/gilliginsisland/pacman/pkg/flagutil"
	"github.com/jessevdk/go-flags"
)

func init() {
	parser.AddCommand("check", "Check host rules", "Check if a host matches the ruleset", &CheckCmd{})
}

var _ flags.Commander = (*CheckCmd)(nil)

// CheckCmd defines the "check" command.
type CheckCmd struct {
	Host      string        `long:"host" required:"true" description:"Host to check"`
	RulesFile flagutil.File `short:"f" long:"file" description:"Path to the rules file" required:"true"`
}

// Execute runs the check command.
func (c *CheckCmd) Execute(args []string) error {
	var rules ghost.RuleSet
	if err := json.NewDecoder(&c.RulesFile).Decode(&rules); err != nil {
		return err
	}
	c.RulesFile.Close()

	if m, ok := rules.Compile().Match(c.Host); ok {
		fmt.Printf("%s\n", m)
		return nil
	}

	os.Exit(1)

	return nil
}
