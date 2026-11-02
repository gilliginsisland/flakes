package dropboxremote

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

type redactedError struct {
	cause   error
	message string
}

func (e *redactedError) Error() string { return e.message }
func (e *redactedError) Unwrap() error { return e.cause }

func commandError(err error, secrets ...string) error {
	if err == nil {
		return nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && len(exit.Stderr) > 0 {
		err = fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exit.Stderr)))
	}
	return redactError(err, secrets...)
}

func redactError(err error, secrets ...string) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if strings.TrimSpace(message) == "" {
		details, marshalErr := json.Marshal(err)
		message = fmt.Sprintf("%T (empty error message)", err)
		if marshalErr == nil && string(details) != "{}" {
			message += ": " + string(details)
		}
	}
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		for _, value := range []string{secret, url.QueryEscape(secret), url.PathEscape(secret)} {
			message = strings.ReplaceAll(message, value, "[redacted]")
		}
	}
	return &redactedError{cause: err, message: message}
}
