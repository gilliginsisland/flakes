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
	params := map[string]any{
		"message":       d.Message,
		"secure":        d.Secure,
		"defaultAnswer": d.DefaultAnswer,
		"title":         d.Title,
		"buttons":       d.Buttons,
		"defaultButton": d.DefaultButton,
		"cancelButton":  d.CancelButton,
	}

	args, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("failed to encode dialog params: %w", err)
	}

	script := fmt.Sprintf(`
		ObjC.import('stdlib');

		var args = %s;
		var app = Application.currentApplication();
		app.includeStandardAdditions = true;

		var options = {
		  defaultAnswer: args.defaultAnswer,
		  hiddenAnswer: args.secure,
		};
		if (args.title) {
		  options.withTitle = args.title;
		}
		if (args.buttons) {
		  buttons = args.buttons;
		}
		if (args.defaultButton) {
		  defaultButton = args.defaultButton;
		}
		if (args.cancelButton) {
		  cancelButton = args.cancelButton;
		}

		try {
			var result = app.displayDialog(args.message, options);
		} catch(e) {
			if (e.message.includes("User canceled")) {
				$.exit(128);
			}
			throw e;
		}
		result.textReturned;
	`, args)

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
