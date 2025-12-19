package notify

import (
	"fmt"
	"sync/atomic"

	"github.com/caseymrm/menuet"

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

// Notify displays a notification and returns a channel that will receive the user's response.
// If Identifier is empty, a unique one is generated automatically.
// If another notification with the same Identifier exists, it is replaced and its channel closed.
func Notify(n Notification, ch chan<- string) {
	// Ensure a unique or provided identifier
	if n.Identifier == "" {
		n.Identifier = fmt.Sprintf("auto-%d", counter.Add(1))
	}

	// Atomically swap out any previous entry
	var (
		prev   chan<- string
		loaded bool
	)
	if ch != nil {
		prev, loaded = chans.Swap(n.Identifier, ch)
	} else {
		prev, loaded = chans.LoadAndDelete(n.Identifier)
	}
	if loaded {
		close(prev)
	}

	// Dispatch notification
	menuet.App().Notification(menuet.Notification(n))
}

func ResponseHandler(id, response string) {
	ch, ok := chans.LoadAndDelete(id)
	if !ok {
		return
	}

	select {
	case ch <- response:
	default:
	}

	close(ch)
}
