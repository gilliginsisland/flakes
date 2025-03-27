package syncutil

import (
	"sync"
)

// Pool manages a synchronized map of objects, created on demand.
type Pool[K comparable, V any] struct {
	mu  sync.RWMutex
	m   map[K]V
	new func(K) (V, error)
}

// NewPool initializes a new Pool with a constructor function.
func NewPool[K comparable, V any](factory func(K) (V, error)) *Pool[K, V] {
	return &Pool[K, V]{
		m:   make(map[K]V),
		new: factory,
	}
}

// Get retrieves an item from the pool, creating it if necessary.
func (p *Pool[K, V]) Get(key K) (V, error) {
	// Fast path: try reading without locking
	p.mu.RLock()
	v, exists := p.m[key]
	p.mu.RUnlock()
	if exists {
		return v, nil
	}

	// Slow path: acquire full lock and check again
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check if another goroutine already created it
	if v, exists = p.m[key]; exists {
		return v, nil
	}

	// Create new instance
	v, err := p.new(key)
	if err != nil {
		var zero V // Return zero value on error
		return zero, err
	}

	p.m[key] = v
	return v, nil
}

// Delete removes an item from the pool.
func (p *Pool[K, V]) Delete(key K) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.m, key)
}
