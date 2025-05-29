//go:build darwin
// +build darwin

package prompt

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var ErrUserCancelled = errors.New("User cancelled")

// prompter implements Prompter interface for macOS.
type prompter struct{}

func (prompter) Prompt(d Dialog) (string, error) {
	args := struct {
		Message string         `json:"message"`
		Options map[string]any `json:"options"`
	}{
		Message: d.Message,
		Options: map[string]any{},
	}

	switch d.Input {
	case SecureInput:
		args.Options["hiddenAnswer"] = true
		fallthrough
	case TextInput:
		args.Options["defaultAnswer"] = d.DefaultAnswer
	}

	if d.Title != "" {
		args.Options["withTitle"] = d.Title
	}
	if d.Buttons != nil {
		args.Options["buttons"] = d.Buttons
	}
	if d.DefaultButton != "" {
		args.Options["defaultButton"] = d.DefaultButton
	}
	if d.CancelButton != "" {
		args.Options["cancelButton"] = d.CancelButton
	}

	jsargs, err := json.Marshal(&args)
	if err != nil {
		return "", fmt.Errorf("failed to encode dialog params: %w", err)
	}

	script := fmt.Sprintf(`
		ObjC.import('stdlib');

		var args = %s;
		var app = Application.currentApplication();
		app.includeStandardAdditions = true;

		try {
			var result = app.displayDialog(args.message, args.options);
		} catch(e) {
			if (e.message.includes("User canceled")) $.exit(128);
			throw e;
		}
		result.textReturned;
	`, jsargs)

	cmd := exec.Command("osascript", "-l", "JavaScript")
	cmd.Stdin = strings.NewReader(script)

	output, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if ee.ExitCode() == 128 {
				err = ErrUserCancelled
			} else {
				err = errors.New(strings.TrimSpace(string(ee.Stderr)))
			}
		}
		return "", fmt.Errorf("prompt failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
