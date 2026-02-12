package syncutil

import (
	"runtime"
	"weak"
)

type cleanupArgs[K comparable, T any] struct {
	k  K
	wp weak.Pointer[T]
}

type WeakMap[K comparable, T any] struct {
	m Map[K, weak.Pointer[T]]
}

// cleanup registers the CompareAndDelete cleanup.
func (m *WeakMap[K, T]) cleanup(args cleanupArgs[K, T]) {
	m.m.CompareAndDelete(args.k, args.wp)
}

func (m *WeakMap[K, T]) Load(key K) *T {
	for {
		wp, loaded := m.m.Load(key)
		if !loaded {
			return nil
		}
		if val := wp.Value(); val != nil {
			return val
		}
		if m.m.CompareAndDelete(key, wp) {
			return nil
		}
	}
}

func (m *WeakMap[K, T]) Store(key K, val *T) {
	if val == nil {
		m.Delete(key)
		return
	}
	wp := weak.Make(val)
	runtime.AddCleanup(val, m.cleanup, cleanupArgs[K, T]{k: key, wp: wp})
	m.m.Store(key, wp)
}

func (m *WeakMap[K, T]) LoadOrStore(key K, newVal *T) *T {
	if newVal == nil {
		return m.Load(key)
	}

	newWp := weak.Make(newVal)

	for {
		currWp, loaded := m.m.LoadOrStore(key, newWp)
		if !loaded {
			// if stored value was used then just return the newVal directly
			runtime.AddCleanup(newVal, m.cleanup, cleanupArgs[K, T]{k: key, wp: newWp})
			return newVal
		}

		if currVal := currWp.Value(); currVal != nil {
			return currVal
		}

		// Value was GC'd between LoadOrStore and now; overwrite it.
		if m.m.CompareAndSwap(key, currWp, newWp) {
			return newVal
		}
	}
}

func (m *WeakMap[K, T]) LoadAndDelete(key K) *T {
	wp, _ := m.m.LoadAndDelete(key)
	return wp.Value()
}

func (m *WeakMap[K, T]) Delete(key K) {
	m.m.Delete(key)
}

func (m *WeakMap[K, T]) Swap(key K, val *T) *T {
	if val == nil {
		return m.LoadAndDelete(key)
	}
	wp := weak.Make(val)
	runtime.AddCleanup(val, m.cleanup, cleanupArgs[K, T]{k: key, wp: wp})
	wp, _ = m.m.Swap(key, wp)
	return wp.Value()
}

func (m *WeakMap[K, T]) CompareAndSwap(key K, old, new *T) bool {
	if new == nil {
		return m.CompareAndDelete(key, old)
	}
	wp := weak.Make(new)
	swapped := m.m.CompareAndSwap(key, weak.Make(old), wp)
	if swapped {
		runtime.AddCleanup(new, m.cleanup, cleanupArgs[K, T]{k: key, wp: wp})
	}
	return swapped
}

func (m *WeakMap[K, T]) CompareAndDelete(key K, old *T) bool {
	if old == nil {
		return false
	}
	return m.m.CompareAndDelete(key, weak.Make(old))
}

func (m *WeakMap[K, T]) Clear() {
	m.m.Clear()
}

// StoreRandom generates a key using the provided generator (e.g., uuid.New)
// and attempts to store the value. It loops until it finds an unused or
// GC'd slot, ensuring the returned key is now mapped to val.
func (m *WeakMap[K, T]) StoreRandom(val *T, keyGen func() K) K {
	if val == nil {
		var zero K
		return zero
	}

	newWp := weak.Make(val)
	for {
		key := keyGen()
		currWp, loaded := m.m.LoadOrStore(key, newWp)
		if !loaded || (currWp.Value() == nil && m.m.CompareAndSwap(key, currWp, newWp)) {
			return key
		}
	}
}
