package notify

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"weak"

	"github.com/caseymrm/menuet"

	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

type Notification = menuet.Notification

var (
	// Map of notification ID → weak pointer to response channel
	chans syncutil.Map[string, weak.Pointer[chan<- string]]
	// Atomic counter for auto-generated IDs
	counter atomic.Uint64
)

func init() {
	menuet.App().NotificationResponder = ResponseHandler
}

// Notify displays a notification and returns a channel that will receive the user's response.
// If Identifier is empty, a unique one is generated automatically.
// If another notification with the same Identifier exists, it is replaced and its channel closed.
func Notify(n Notification) <-chan string {
	// Ensure a unique or provided identifier
	if n.Identifier == "" {
		n.Identifier = fmt.Sprintf("auto-%d", counter.Add(1))
	}

	// Create a new channel for responses
	ch := make(chan string, 1)
	rch := (chan<- string)(ch)
	wp := weak.Make(&rch)

	// Atomically swap out any previous entry
	if prev, loaded := chans.Swap(n.Identifier, wp); loaded {
		// Close previous channel if still alive
		if ch := prev.Value(); ch != nil {
			close(*ch)
		}
	}

	// Set up automatic cleanup
	runtime.AddCleanup(&ch, func(id string) {
		chans.CompareAndDelete(id, wp)
	}, n.Identifier)

	// Dispatch notification
	menuet.App().Notification(n)

	return ch
}

func ResponseHandler(id, response string) {
	wp, ok := chans.LoadAndDelete(id)
	if !ok {
		return
	}

	ch := wp.Value()
	if ch == nil {
		return
	}

	select {
	case *ch <- response:
	default:
	}

	close(*ch)
}
