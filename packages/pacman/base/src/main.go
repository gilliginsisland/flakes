package main

import (
	"os"

	"github.com/gilliginsisland/pacman/cmd"
)

func main() {
	// Parse and execute subcommands
	_, err := cmd.ParseArgs(os.Args[1:])
	if err != nil {
		os.Exit(1)
	}
}
