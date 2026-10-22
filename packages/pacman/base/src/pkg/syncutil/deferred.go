package syncutil

import "context"

// Deferred holds the result of a function running in its own goroutine.
// It must not be copied after construction.
type Deferred[T1, T2 any] struct {
	done   chan struct{}
	value  values[T1, T2]
	cancel context.CancelCauseFunc
}

// NewDeferred starts fn with a child of ctx and cancels it after publishing the
// result. Done closes only after fn returns, even if ctx is canceled earlier.
// A returned resource must not rely on this setup context remaining live.
func NewDeferred[T1, T2 any](ctx context.Context, fn func(context.Context) (T1, T2)) *Deferred[T1, T2] {
	ctx, cancel := context.WithCancelCause(ctx)
	d := Deferred[T1, T2]{
		done:   make(chan struct{}),
		cancel: cancel,
	}
	go func() {
		defer d.cancel(nil)
		d.value.v1, d.value.v2 = fn(ctx)
		close(d.done)
	}()
	return &d
}

// Cancel cancels the function's context without waiting for it to return.
func (d *Deferred[T1, T2]) Cancel(cause error) {
	d.cancel(cause)
}

// Done closes when the result is available. A nil Deferred returns a nil channel.
func (d *Deferred[T1, T2]) Done() <-chan struct{} {
	if d == nil {
		return nil
	}
	return d.done
}

// Values returns the result without blocking, or zero values if not done.
func (d *Deferred[T1, T2]) Values() (v1 T1, v2 T2) {
	select {
	case <-d.Done():
		return d.value.v1, d.value.v2
	default:
		return
	}
}
