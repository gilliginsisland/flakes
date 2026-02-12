package xdg

import (
	"os/exec"
)

func open(input string) *exec.Cmd {
	return exec.Command("open", input)
}

func openWith(input string, app string) *exec.Cmd {
	return exec.Command("open", "-a", app, input)
}
