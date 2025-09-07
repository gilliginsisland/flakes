package syncutil

import "sync/atomic"

// AtomicValue is a generic wrapper around sync/atomic.Value for type-safe atomic operations.
type AtomicValue[T any] struct {
    atomic.Value
}

// Store atomically stores a value of type T.
func (av *AtomicValue[T]) Store(val T) {
    av.Value.Store(val)
}

// Load atomically loads the stored value as type T, returning the zero value of T if no value is stored.
func (av *AtomicValue[T]) Load() T {
    if val := av.Value.Load(); val != nil {
        return val.(T)
    }
    var zero T
    return zero
}

// CompareAndSwap atomically swaps the old value with the new value if they match, returning true if successful.
func (av *AtomicValue[T]) CompareAndSwap(oldVal, newVal T) bool {
    return av.Value.CompareAndSwap(oldVal, newVal)
}
