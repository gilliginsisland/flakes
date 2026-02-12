package syncutil

import (
    "sync"
)

// Signal manages state change notifications for a generic state type T with a separate lock.
type Signal[T any] struct {
    mu  sync.Mutex
    chs []chan<- T
}

// Subscribe returns a receive-only channel for state change notifications
// and a cancellation function to stop receiving updates.
func (s *Signal[T]) Subscribe() (<-chan T, func()) {
    s.mu.Lock()
    defer s.mu.Unlock()

    ch := make(chan T, 10) // Buffered to avoid blocking sender
    index := len(s.chs)
    s.chs = append(s.chs, ch)

    cancel := func() {
        s.mu.Lock()
        defer s.mu.Unlock()
        // Start from the minimum of index or the current end of slice
        for i := min(index, len(s.chs)-1); i >= 0; i-- {
            if s.chs[i] != ch {
                continue
            }
            s.chs = append(s.chs[:i], s.chs[i+1:]...)
            close(ch)
            break
        }
    }

    return ch, cancel
}

// Publish notifies all subscribed channels of a state change.
func (s *Signal[T]) Publish(msg T) {
    s.mu.Lock()
    defer s.mu.Unlock()
    for _, ch := range s.chs {
        select {
        case ch <- msg:
        default:
            // Drop if channel is full to avoid blocking
        }
    }
}
