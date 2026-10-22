package syncutil

import "context"

// Discard receives count messages without inspecting their values. It returns
// the number received, stopping early if the context is canceled or ch closes.
func Discard[T any](ctx context.Context, ch <-chan T, count uint64) (n uint64) {
	for n < count {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-ctx.Done():
			return
		}
		n++
	}
	return n
}
