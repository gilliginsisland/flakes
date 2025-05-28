//go:build darwin
// +build darwin

package notify

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// notifier is the darwin implementation using AppleScript.
type notifier struct{}

func (notifier) Send(n Notification) error {
	if n.Title == "" {
		n.Title = "PacMan"
	}

	args, err := json.Marshal(map[string]string{
		"title":    n.Title,
		"message":  n.Message,
		"subtitle": n.Subtitle,
		"sound":    n.SoundName,
	})
	if err != nil {
		return fmt.Errorf("failed to encode notification params: %w", err)
	}

	script := fmt.Sprintf(`
		var args = %s;
		var app = Application.currentApplication();
		app.includeStandardAdditions = true;

		app.displayNotification(args.message, {
		  withTitle: args.title,
		  subtitle: args.subtitle,
		  soundName: args.sound
		});
	`, args)

	cmd := exec.Command("osascript", "-l", "JavaScript")
	cmd.Stdin = strings.NewReader(script)

	err = cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			err = errors.New(strings.TrimSpace(string(ee.Stderr)))
		}
		return fmt.Errorf("notify failed: %w", err)
	}

	return nil
}
