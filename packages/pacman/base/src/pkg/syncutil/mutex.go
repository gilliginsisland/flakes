package syncutil

import (
	"context"
	"sync"
)

// CtxLocker is a lock whose acquisition can be cancelled by a context.
// A successful acquisition must be paired with Unlock.
type CtxLocker interface {
	sync.Locker
	LockCtx(context.Context) bool
	TryLock() bool
}

func NewCtxLocker() CtxLocker {
	return make(ctxLocker, 1)
}

type ctxLocker chan struct{}

func (m ctxLocker) Lock() {
	m.LockCtx(context.Background())
}

func (m ctxLocker) LockCtx(ctx context.Context) bool {
	select {
	case m <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (m ctxLocker) TryLock() bool {
	select {
	case m <- struct{}{}:
		return true
	default:
		return false
	}
}

func (m ctxLocker) Unlock() {
	select {
	case <-m:
	default:
		panic("syncutil: unlock of unlocked CtxLocker")
	}
}
