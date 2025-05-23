//go:build !darwin
// +build !darwin

package notify

import "fmt"

// notifier for non-darwin platforms.
type notifier struct{}

func (notifier) Send(n Notification) error {
	return fmt.Errorf("notifications are not supported on this platform")
}
