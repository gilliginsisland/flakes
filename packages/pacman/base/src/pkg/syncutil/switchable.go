package syncutil

import "sync/atomic"

// Switchable controls whether Current returns its stored value or the zero value.
// It must not be copied after first use.
type Switchable[T any] struct {
	value   T
	enabled atomic.Bool
}

// NewSwitchable wraps value, initially disabled.
func NewSwitchable[T any](value T) *Switchable[T] {
	s := Switchable[T]{value: value}
	return &s
}

func (s *Switchable[T]) Enable() {
	s.enabled.Store(true)
}

func (s *Switchable[T]) Disable() {
	s.enabled.Store(false)
}

// Current returns the zero value when disabled. Enable and Disable do not wake an
// existing select using a previously returned channel.
func (s *Switchable[T]) Current() (value T) {
	if s.enabled.Load() {
		return s.value
	}
	return
}

// Value returns the stored value regardless of whether it is enabled.
func (s *Switchable[T]) Value() T {
	return s.value
}
