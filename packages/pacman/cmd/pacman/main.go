package main

import (
	"os"
)

func main() {
	args := os.Args

	// for csd wrapper, there is no way to pass extra args
	// also the flags are sent in an incompatible way
	if len(args) > 1 && args[1] == "" {
		if _, ok := os.LookupEnv("CSD_TOKEN"); ok {
			args = []string{"csd"}
		}
	} else {
		args = args[1:]
	}

	// Parse and execute subcommands
	_, err := parser.ParseArgs(args)
	if err != nil {
		os.Exit(1)
	}
}
