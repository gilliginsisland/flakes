package pool

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

var errExpired = errors.New("expired")

type entry[V any] struct {
	val     V
	err     error
	refs    chan context.Context
	expired chan struct{}
}

func (f *entry[V]) Unwrap(ctx context.Context) (V, error) {
	select {
	case f.refs <- ctx:
		return f.val, f.err
	case <-f.expired:
		if f.err != nil {
			return f.val, f.err
		}
		return f.val, errExpired
	}
}

type Pool[K comparable, V any] struct {
	mu      sync.RWMutex
	items   map[K]*entry[V]
	factory func(K) (V, error)
	timeout func(K) <-chan time.Time
}

func New[K comparable, V any](
	factory func(K) (V, error),
	timeout func(K) <-chan time.Time,
) *Pool[K, V] {
	return &Pool[K, V]{
		items:   make(map[K]*entry[V]),
		factory: factory,
		timeout: timeout,
	}
}

func (p *Pool[K, V]) GetCtx(ctx context.Context, key K) (V, error) {
	for {
		item := p.get(key)
		val, err := item.Unwrap(ctx)
		if err != errExpired {
			return val, err
		}
	}
}

func (p *Pool[K, V]) get(key K) *entry[V] {
	// Fast path: try reading without locking
	p.mu.RLock()
	item, exists := p.items[key]
	p.mu.RUnlock()

	if exists {
		return item
	}

	// Slow path: acquire full lock and check again
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check if another goroutine already created it
	item, exists = p.items[key]
	if exists {
		return item
	}

	// Use a future so we can generate the item async
	item = &entry[V]{
		refs:    make(chan context.Context),
		expired: make(chan struct{}),
	}
	p.items[key] = item

	go func() {
		item.val, item.err = p.factory(key)

		if item.err != nil {
			p.mu.Lock()
			delete(p.items, key)
			p.mu.Unlock()
			return
		}

		go p.monitor(key, item)
	}()

	return item
}

func (p *Pool[K, V]) monitor(key K, item *entry[V]) {
	var (
		wait     chan struct{}
		timeout  <-chan time.Time
		done     = make(chan struct{})
		refCount int
	)

	if w, ok := any(item.val).(interface{ Wait() error }); ok {
		wait = make(chan struct{})
		go func() {
			w.Wait()
			close(wait)
		}()
	}

	for {
		select {
		case ctx := <-item.refs:
			refCount++
			timeout = nil

			go func() {
				select {
				case <-ctx.Done():
					select {
					case done <- struct{}{}:
					case <-item.expired:
					}
				case <-item.expired:
				}
			}()
		case <-done:
			refCount--
			if refCount == 0 {
				timeout = p.timeout(key)
			}
		case <-wait:
			p.mu.Lock()
			delete(p.items, key)
			p.mu.Unlock()
			close(item.expired)
			return
		case <-timeout:
			p.mu.Lock()
			delete(p.items, key)
			p.mu.Unlock()
			close(item.expired)
			if c, ok := any(item.val).(io.Closer); ok {
				go c.Close()
			}
			return
		}
	}
}
