package syncutil

import (
	"sync"
)

// ParallelRange returns a function compatible with Go's range-over-func
// It executes each iteration in a goroutine and manages a WaitGroup automatically.
func ParallelRange[T any](items []T) func(yield func(T) bool) {
	return func(yield func(T) bool) {
		var wg sync.WaitGroup
		for _, item := range items {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if !yield(item) {
					return
				}
			}()
		}
		wg.Wait()
	}
}
