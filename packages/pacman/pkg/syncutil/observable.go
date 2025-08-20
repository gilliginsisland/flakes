package syncutil

import "sync/atomic"

// Observable is an atomic value with change notifications.
type Observable[T any] struct {
	val    atomic.Value
	signal Signal[func() T]
}

// Store sets a new value atomically and notifies observers.
func (o *Observable[T]) Store(v T) {
	o.val.Store(v)
	o.signal.Signal(o.Load)
}

// Load retrieves the current value and a bool indicating presence.
func (o *Observable[T]) Load() T {
	raw := o.val.Load()
	if raw == nil {
		var zero T
		return zero
	}
	return raw.(T)
}

// Observer returns a channel of loader functions.
// On every Store, a loader is sent which can be called to fetch the latest value.
func (o *Observable[T]) Observe() <-chan func() T {
	return o.signal.Receive()
}

// Close closes all observers.
func (o *Observable[T]) Close() {
	o.signal.Close()
}
