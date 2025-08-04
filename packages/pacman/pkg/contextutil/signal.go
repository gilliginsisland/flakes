package contextutil

import (
	"context"
)

// Merge returns a context derived from base, and is cancelled when either
// base or other is cancelled. The cancel cause (if any) is preserved.
func Merge(base, other context.Context) context.Context {
	ctx, cancel := context.WithCancelCause(base)

	go func() {
		select {
		case <-base.Done():
		case <-other.Done():
			cancel(context.Cause(other))
		}
	}()

	return ctx
}
