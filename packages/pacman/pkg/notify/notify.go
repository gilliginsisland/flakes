package notify

// Notification represents a system notification.
type Notification struct {
	Message   string
	Title     string
	Subtitle  string
	SoundName string
}

// Notifier defines an interface for sending notifications.
type Notifier interface {
	Send(n Notification) error
}

// New returns a platform-specific Notifier implementation.
func New() Notifier {
	return notifier{}
}
