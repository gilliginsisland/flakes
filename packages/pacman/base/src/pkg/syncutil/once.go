package syncutil

import "sync"

// Once extends sync.Once with a Go method that runs the function in a goroutine.
type Once struct {
	sync.Once // Embedded sync.Once for standard Do behavior
}

// Go executes the function f in a goroutine if it hasn't been done before.
// Subsequent calls return immediately without waiting for f to complete.
func (o *Once) Go(f func()) bool {
	return o.Do(func() {
		go f()
	})
}

func (o *Once) Do(f func()) bool {
	var ran bool
	o.Once.Do(func() {
		ran = true
		f()
	})
	return ran
}
