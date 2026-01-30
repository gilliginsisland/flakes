package syncutil

import "sync"

type Map[K comparable, V any] struct {
	sync.Map
}

func (m *Map[K, V]) Store(key K, value V) {
	m.Map.Store(key, value)
}

func (m *Map[K, V]) Load(key K) (V, bool) {
	return m.zero(m.Map.Load(key))
}

func (m *Map[K, V]) Delete(key K) {
	m.Map.Delete(key)
}

func (m *Map[K, V]) LoadOrStore(key K, value V) (V, bool) {
	return m.zero(m.Map.LoadOrStore(key, value))
}

func (m *Map[K, V]) LoadAndDelete(key K) (V, bool) {
	return m.zero(m.Map.LoadAndDelete(key))
}

func (m *Map[K, V]) Swap(key K, value V) (V, bool) {
	return m.zero(m.Map.Swap(key, value))
}

func (m *Map[K, V]) CompareAndDelete(key K, old V) bool {
	return m.Map.CompareAndDelete(key, old)
}

func (m *Map[K, V]) CompareAndSwap(key K, old, new V) bool {
	return m.Map.CompareAndSwap(key, old, new)
}

func (m *Map[K, V]) zero(val any, loaded bool) (V, bool) {
	if val != nil {
		return val.(V), loaded
	}
	if loaded {
		// this logically this cannot happen. any value stored in the map would be non nil.
		// even a nil would be typed and not nil.
		panic("loaded value cannot be nil")
	}
	var zero V
	return zero, false
}
