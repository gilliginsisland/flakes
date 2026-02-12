package syncutil

import (
    "sync"
    "sync/atomic"
    "time"
)

// Timeout represents a stoppable timeout that executes a callback under a provided lock.
type Timeout struct {
    locker   sync.Locker
    timer    *time.Timer
    deadline atomic.Uint64 // Stores the deadline as UnixNano for the current timeout, 0 means stopped
    fn       func()
}

// NewTimeout creates a new Timeout with a given locker, duration, and callback function to run on timeout.
func NewTimeout(locker sync.Locker, d time.Duration, fn func()) *Timeout {
    t := &Timeout{
        locker: locker,
        fn:     fn,
    }
    deadline := uint64(time.Now().Add(d).UnixNano())
    t.deadline.Store(deadline)
    t.timer = time.AfterFunc(d, t.runWithLock)
    return t
}

// runWithLock attempts to acquire the lock and run the callback if deadline hasn't been updated or stopped.
func (t *Timeout) runWithLock() {
    t.locker.Lock()
    defer t.locker.Unlock()

    if d := t.deadline.Load(); d == 0 || d > uint64(time.Now().UnixNano()) {
        // don't run the callback if stopped (deadline=0)
        // or a newer deadline is set (via Reset)
        return
    }

    t.fn()
}

// Stop stops the timeout, potentially preventing the callback from executing.
// This is only garaunteed if called under a lock.
func (t *Timeout) Stop() {
    t.timer.Stop()
    // Set deadline to 0 to indicate stopped state.
    t.deadline.Store(0)
}

// Reset resets the timeout to a new duration, potentially preventing the callback
// from executing before the new duration. This is only garaunteed if called under a lock.
func (t *Timeout) Reset(d time.Duration) {
    deadline := uint64(time.Now().Add(d).UnixNano())
    t.deadline.Store(deadline)
    t.timer.Reset(d)
}
