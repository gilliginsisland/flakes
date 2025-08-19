package main

import (
	"encoding/json"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/gilliginsisland/pacman/pkg/proxy"
	"github.com/gilliginsisland/pacman/pkg/trie"
)

func init() {
	parser.AddCommand("check", "Check host rules", "Check if a host matches the ruleset", &CheckCmd{})
}

var _ flags.Commander = (*CheckCmd)(nil)

// CheckCmd defines the "check" command.
type CheckCmd struct{}

// Execute runs the check command.
func (c *CheckCmd) Execute(args []string) error {
	var rs proxy.RuleSet
	if err := json.NewDecoder(&opts.RulesFile).Decode(&rs); err != nil {
		return err
	}
	opts.RulesFile.Close()

	t := trie.NewHost[string]()
	for _, rule := range rs.Rules {
		for _, host := range rule.Hosts {
			t.Insert(host, host)
		}
	}

	for _, host := range args {
		if _, ok := t.Match(host); ok {
			return nil
		}
	}

	os.Exit(1)

	return nil
}
