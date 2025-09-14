package throttle

import (
	"time"
)

// Counter tracks counts of a generic value within a sliding time window for throttling purposes.
type Counter[T any] struct {
	MaxCount   int           // Maximum allowed count before throttling
	Window     time.Duration // Sliding time window to count items
	lastValue  T             // Last value encountered
	timestamps []time.Time   // Timestamps of recorded events for sliding window
}

// Increment records a new value and checks if throttling is needed.
func (c *Counter[T]) Increment(value T) bool {
	now := time.Now()
	// Prune timestamps outside the sliding window
	c.prune(now)

	// Append the current timestamp
	c.timestamps = append(c.timestamps, now)
	c.lastValue = value

	// Return true if throttling is needed (count exceeds max)
	return len(c.timestamps) > c.MaxCount
}

// prune removes timestamps older than the sliding window.
func (c *Counter[T]) prune(now time.Time) {
	cutoff := now.Add(-c.Window)
	// Find the first timestamp within the window
	cutIndex := 0
	for i, ts := range c.timestamps {
		if ts.After(cutoff) {
			cutIndex = i
			break
		}
	}
	// If cutIndex is not 0, slice the timestamps to keep only those within the window
	if cutIndex > 0 {
		c.timestamps = c.timestamps[cutIndex:]
	}
}

// Count returns the current number of items within the sliding window.
func (c *Counter[T]) Count() int {
	c.prune(time.Now())
	return len(c.timestamps)
}

// Reset clears the count and last value, typically called on success.
func (c *Counter[T]) Reset() {
	c.timestamps = c.timestamps[:0] // Clear slice but preserve capacity
	var zero T
	c.lastValue = zero
}

// Throttled checks if the current count exceeds the maximum allowed within the sliding window.
func (c *Counter[T]) Throttled() bool {
	c.prune(time.Now())
	return len(c.timestamps) > c.MaxCount
}

// Force sets the state to indicate throttling is active, as if the max count was exceeded.
func (c *Counter[T]) Force(value T) {
	c.lastValue = value // Store the value (e.g., cancellation error)
	// Clear existing timestamps but preserve capacity
	c.timestamps = c.timestamps[:0]
	// Append timestamps to maintain the throttled state for the window duration
	now := time.Now()
	for i := 0; i <= c.MaxCount; i++ {
		c.timestamps = append(c.timestamps, now)
	}
}

// Last returns the most recent value and its associated timestamp, if available.
func (c *Counter[T]) Last() (T, time.Time, bool) {
	c.prune(time.Now())
	if len(c.timestamps) == 0 {
		var zero T
		return zero, time.Time{}, false
	}
	return c.lastValue, c.timestamps[len(c.timestamps)-1], true
}
