package syncutil

import "sync/atomic"

// AtomicValues atomically stores and loads a pair of values. Its zero value
// contains the zero values of T1 and T2. It must not be copied after first use.
type AtomicValues[T1, T2 any] struct {
	value atomic.Pointer[values[T1, T2]]
}

// Store atomically replaces both values.
func (av *AtomicValues[T1, T2]) Store(v1 T1, v2 T2) {
	value := values[T1, T2]{v1: v1, v2: v2}
	av.value.Store(&value)
}

// Load atomically returns both values from the same Store.
// Before the first Store, it returns their zero values.
func (av *AtomicValues[T1, T2]) Load() (v1 T1, v2 T2) {
	if value := av.value.Load(); value != nil {
		return value.v1, value.v2
	}
	return
}
