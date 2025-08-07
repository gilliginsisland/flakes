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
