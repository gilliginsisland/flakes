package notify

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/gilliginsisland/pacman/pkg/menuet/notifications"
	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

// Notification is the Go convenience input used to build native notification
// content and an immediate delivery request.
type Notification struct {
	CategoryIdentifier  string
	Identifier          string
	Title               string
	Subtitle            string
	Body                string
	PresentationOptions notifications.PresentationOptions
	InterruptionLevel   notifications.InterruptionLevel
}

type Response struct {
	NotificationIdentifier string
	ActionIdentifier       string
	Text                   string
}

var (
	// Map of notification ID → response channel
	chans syncutil.Map[string, chan<- Response]
	// Atomic counter for auto-generated IDs
	counter atomic.Uint64
)

// Notify displays a notification.
// If Identifier is empty, a unique one is generated automatically.
// If Identifier is set, any pending or delivered notification with that ID is removed before display.
func Notify(n Notification) string {
	if n.Title == "" {
		n.Title = "PACman"
	}
	id, _ := notify(n, nil)
	return id
}

// NotifyCh displays a notification and returns a channel that will receive the user's response.
// If Identifier is empty, a unique one is generated automatically.
// If Identifier is set, any pending or delivered notification with that ID is removed before display.
// If another response channel has the same Identifier, it is closed.
// A cleanup function is provided to discard the channel and remove the notification.
func NotifyCh(n Notification) (<-chan Response, func()) {
	ch := make(chan Response, 1)
	_, cleanup := notify(n, ch)
	return ch, cleanup
}

func NotifyCtx(ctx context.Context, n Notification) (Response, error) {
	ch, cleanup := NotifyCh(n)
	select {
	case <-ctx.Done():
		defer cleanup()
		return Response{}, ctx.Err()
	case resp := <-ch:
		return resp, nil
	}
}

func notify(n Notification, ch chan<- Response) (string, func()) {
	// Ensure a unique or provided identifier
	if n.Identifier == "" {
		n.Identifier = fmt.Sprintf("auto-%d", counter.Add(1))
	} else {
		Remove(n.Identifier)
	}
	if n.PresentationOptions == notifications.PresentationOptionNone {
		n.PresentationOptions =
			notifications.PresentationOptionBadge |
				notifications.PresentationOptionSound |
				notifications.PresentationOptionList |
				notifications.PresentationOptionBanner
	}

	// Atomically swap out any previous entry
	var (
		prev    chan<- Response
		loaded  bool
		cleanup func()
	)
	if ch != nil {
		prev, loaded = chans.Swap(n.Identifier, ch)
		cleanup = func() {
			if chans.CompareAndDelete(n.Identifier, ch) {
				close(ch)
				Remove(n.Identifier)
			}
		}
	} else {
		prev, loaded = chans.LoadAndDelete(n.Identifier)
	}
	if loaded {
		close(prev)
	}

	// Dispatch notification
	content := notifications.NewContent().
		SetCategoryIdentifier(n.CategoryIdentifier).
		SetTitle(n.Title).
		SetSubtitle(n.Subtitle).
		SetBody(n.Body).
		SetPresentationOptions(n.PresentationOptions).
		SetInterruptionLevel(n.InterruptionLevel)
	notifications.CurrentCenter().AddRequest(notifications.NewRequest(notifications.RequestOptions{
		Identifier: n.Identifier,
		Content:    content,
	}))

	return n.Identifier, cleanup
}

func ResponseHandler(response *notifications.Response) {
	resp := Response{
		NotificationIdentifier: response.NotificationIdentifier(),
		ActionIdentifier:       response.ActionIdentifier(),
		Text:                   response.Text(),
	}
	defer Remove(resp.NotificationIdentifier)

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

// Remove removes pending and delivered notifications by identifier.
func Remove(identifiers ...string) {
	for _, identifier := range identifiers {
		if identifier == "" {
			continue
		}
		center := notifications.CurrentCenter()
		center.RemovePendingRequests(identifier)
		center.RemoveDeliveredNotifications(identifier)
	}
}
