package syncutil

import "sync"

// Signal notifies subscribers of events.
type Signal[T any] struct {
	mu   sync.Mutex
	subs []chan<- T
}

// Receive returns a channel that will get a notification when Send is called.
// The channel is buffered with capacity 1, so only the most recent signal matters.
func (s *Signal[T]) Receive() <-chan T {
	ch := make(chan T, 1)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subs = append(s.subs, ch)
	return ch
}

// Signal notifies all receivers of a change.
// If a channel already has a signal queued, it is skipped.
func (s *Signal[T]) Signal(v T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.subs {
		select {
		case ch <- v:
		default: // already has a signal queued
		}
	}
}

// Close closes all receiver channels.
func (s *Signal[T]) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.subs {
		close(ch)
	}
	s.subs = nil
}
