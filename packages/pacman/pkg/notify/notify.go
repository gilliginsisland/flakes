package notify

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/gilliginsisland/pacman/pkg/menuet"
	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

type Notification = menuet.Notification

var (
	// Map of notification ID → weak pointer to response channel
	chans syncutil.Map[string, chan<- menuet.NotificationResponse]
	// Atomic counter for auto-generated IDs
	counter atomic.Uint64
)

func init() {
	menuet.App().NotificationResponder = ResponseHandler
}

// Notify displays a notification.
// If Identifier is empty, a unique one is generated automatically.
// If another notification with the same Identifier exists, it is replaced.
func Notify(n Notification) {
	if n.Title == "" {
		n.Title = "PACman"
	}
	notify(n, nil)
}

// NotifyCh displays a notification and returns a channel that will receive the user's response.
// If Identifier is empty, a unique one is generated automatically.
// If another notification with the same Identifier exists, it is replaced and its channel closed.
// A cleanup function is provided to discard the channel
func NotifyCh(n Notification) (<-chan menuet.NotificationResponse, func()) {
	ch := make(chan menuet.NotificationResponse, 1)
	return ch, notify(n, ch)
}

func NotifyCtx(ctx context.Context, n Notification) (menuet.NotificationResponse, error) {
	ch, cleanup := NotifyCh(n)
	select {
	case <-ctx.Done():
		defer cleanup()
		return menuet.NotificationResponse{}, ctx.Err()
	case resp := <-ch:
		return resp, nil
	}
}

func notify(n Notification, ch chan<- menuet.NotificationResponse) func() {
	// Ensure a unique or provided identifier
	if n.Identifier == "" {
		n.Identifier = fmt.Sprintf("auto-%d", counter.Add(1))
	}

	// Atomically swap out any previous entry
	var (
		prev    chan<- menuet.NotificationResponse
		loaded  bool
		cleanup func()
	)
	if ch != nil {
		prev, loaded = chans.Swap(n.Identifier, ch)
		cleanup = func() {
			if chans.CompareAndDelete(n.Identifier, ch) {
				close(ch)
			}
		}
	} else {
		prev, loaded = chans.LoadAndDelete(n.Identifier)
	}
	if loaded {
		close(prev)
	}

	// Dispatch notification
	menuet.App().Notification(n)

	return cleanup
}

func ResponseHandler(resp menuet.NotificationResponse) {
	ch, ok := chans.LoadAndDelete(resp.NotificationIdentifier)
	if !ok {
		return
	}
	defer close(ch)

	select {
	case ch <- resp:
	default:
		return
	}
}
