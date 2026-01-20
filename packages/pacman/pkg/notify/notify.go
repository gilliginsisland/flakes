package notify

import (
	"fmt"
	"sync/atomic"

	"github.com/gilliginsisland/pacman/pkg/menuet"
	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

type Notification menuet.Notification

var (
	// Map of notification ID → weak pointer to response channel
	chans syncutil.Map[string, chan<- string]
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
	notify(n, nil)
}

// WithChannel displays a notification and returns a channel that will receive the user's response.
// If Identifier is empty, a unique one is generated automatically.
// If another notification with the same Identifier exists, it is replaced and its channel closed.
// A cleanup function is provided to discard the channel
func WithChannel(n Notification) (<-chan string, func()) {
	ch := make(chan string, 1)
	return ch, notify(n, ch)
}

func notify(n Notification, ch chan<- string) func() {
	// Ensure a unique or provided identifier
	if n.Identifier == "" {
		n.Identifier = fmt.Sprintf("auto-%d", counter.Add(1))
	}

	// Atomically swap out any previous entry
	var (
		prev    chan<- string
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
	menuet.App().Notification(menuet.Notification(n))

	return cleanup
}

func ResponseHandler(resp menuet.NotificationResponse) {
	ch, ok := chans.LoadAndDelete(resp.NotificationIdentifier)
	if !ok {
		return
	}

	select {
	case ch <- resp.Text:
	default:
	}

	close(ch)
}
