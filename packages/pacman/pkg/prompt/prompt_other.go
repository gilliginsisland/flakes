//go:build !darwin
// +build !darwin

package prompt

import "fmt"

type prompter struct{}

func (prompter) Prompt(d Dialog) (string, error) {
	return "", fmt.Errorf("prompt not supported on this platform")
}
