package syncutil

import "sync"

// Once extends sync.Once with a Go method that runs the function in a goroutine.
type Once struct {
	sync.Once // Embedded sync.Once for standard Do behavior
}

// Go executes the function f in a goroutine if it hasn't been done before.
// Subsequent calls return immediately without waiting for f to complete.
func (o *Once) Go(f func()) {
	o.Once.Do(func() {
		go f()
	})
}
