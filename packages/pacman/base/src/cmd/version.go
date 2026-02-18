package cmd

import (
	"fmt"

	"github.com/jessevdk/go-flags"

	"github.com/gilliginsisland/pacman/internal/version"
)

func init() {
	parser.AddCommand("version", "PACman version", "Prints the PACman version", &VersionCmd{})
}

var _ flags.Commander = (*VersionCmd)(nil)

// CheckCmd defines the "check" command.
type VersionCmd struct{}

// Execute runs the check command.
func (c *VersionCmd) Execute(args []string) error {
	fmt.Println(version.Version)
	return nil
}
