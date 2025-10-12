package syncutil

import "sync"

type Map[K comparable, V any] struct {
	sync.Map
}

func (m *Map[K, V]) Store(key K, value V) {
	m.Map.Store(key, value)
}

func (m *Map[K, V]) Load(key K) (V, bool) {
	val, ok := m.Map.Load(key)
	if !ok {
		var zero V
		return zero, false
	}
	return val.(V), true
}

func (m *Map[K, V]) Delete(key K) {
	m.Map.Delete(key)
}

func (m *Map[K, V]) LoadOrStore(key K, value V) (V, bool) {
	actual, loaded := m.Map.LoadOrStore(key, value)
	return actual.(V), loaded
}

func (m *Map[K, V]) LoadAndDelete(key K) (V, bool) {
	val, loaded := m.Map.LoadAndDelete(key)
	if !loaded {
		var zero V
		return zero, false
	}
	return val.(V), true
}

func (m *Map[K, V]) Swap(key K, value V) (V, bool) {
	prev, loaded := m.Map.Swap(key, value)
	if !loaded {
		var zero V
		return zero, false
	}
	return prev.(V), true
}

func (m *Map[K, V]) CompareAndDelete(key K, old V) bool {
	return m.Map.CompareAndDelete(key, old)
}

func (m *Map[K, V]) CompareAndSwap(key K, old, new V) bool {
	return m.Map.CompareAndSwap(key, old, new)
}
