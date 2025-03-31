package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gilliginsisland/pacman/pkg/dialer"
)

func init() {
	parser.AddCommand("check", "Check host rules", "Check if a host matches the ruleset", &CheckCmd{})
}

// CheckCmd defines the "check" command.
type CheckCmd struct {
	Host string `long:"host" required:"true" description:"Host to check"`
}

// Execute runs the check command.
func (c *CheckCmd) Execute(args []string) error {
	var rules dialer.Ruleset
	if err := json.NewDecoder(&opts.RulesFile).Decode(&rules); err != nil {
		return err
	}
	opts.RulesFile.Close()

	if r := rules.MatchHost(c.Host); r != nil {
		fmt.Printf("%s\n", r.Proxies)
		return nil
	}

	os.Exit(1)

	return nil
}
