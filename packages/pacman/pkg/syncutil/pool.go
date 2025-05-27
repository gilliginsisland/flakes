package syncutil

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

var errExpired = errors.New("expired")

type wrapper[V any] interface {
	Unwrap(ctx context.Context) (V, error)
}

type pooled[V any] struct {
	val  V
	refs chan context.Context
	done chan struct{}
}

func (p *pooled[V]) Unwrap(ctx context.Context) (V, error) {
	select {
	case p.refs <- ctx:
		return p.val, nil
	case <-p.done:
		return p.val, errExpired
	}
}

type future[V any] struct {
	val  V
	err  error
	refs chan context.Context
	done chan struct{}
}

func (f *future[V]) Unwrap(ctx context.Context) (V, error) {
	<-f.done
	if f.err == nil {
		f.refs <- ctx
	}
	return f.val, f.err
}

type Pool[K comparable, V any] struct {
	mu      sync.RWMutex
	items   map[K]wrapper[V]
	factory func(K) (V, error)
	timeout time.Duration
}

func NewPool[K comparable, V any](factory func(K) (V, error), timeout time.Duration) *Pool[K, V] {
	return &Pool[K, V]{
		items:   make(map[K]wrapper[V]),
		factory: factory,
		timeout: timeout,
	}
}

func (p *Pool[K, V]) GetCtx(ctx context.Context, key K) (V, error) {
	// Fast path: try reading without locking
	p.mu.RLock()
	item, exists := p.items[key]
	p.mu.RUnlock()

	if exists {
		val, err := item.Unwrap(ctx)
		if err == errExpired {
			return p.GetCtx(ctx, key)
		}
		return val, err
	}

	// Slow path: acquire full lock and check again
	p.mu.Lock()

	// Double-check if another goroutine already created it
	if item, exists = p.items[key]; exists {
		p.mu.Unlock()
		val, err := item.Unwrap(ctx)
		if err == errExpired {
			return p.GetCtx(ctx, key)
		}
		return val, err
	}

	// Use a future so we can generate the item async
	fut := future[V]{
		done: make(chan struct{}),
		refs: make(chan context.Context),
	}
	p.items[key] = &fut
	p.mu.Unlock()

	go func() {
		fut.val, fut.err = p.factory(key)
		close(fut.done)

		if fut.err != nil {
			p.mu.Lock()
			delete(p.items, key)
			p.mu.Unlock()
			return
		}

		item := pooled[V]{
			val:  fut.val,
			refs: fut.refs,
		}
		p.mu.Lock()
		p.items[key] = &item
		p.mu.Unlock()
		go p.monitor(key, &item)
	}()

	return fut.Unwrap(ctx)
}

func (p *Pool[K, V]) monitor(key K, item *pooled[V]) {
	var (
		wait     chan error
		timeout  <-chan time.Time
		done     chan struct{}
		refCount int
	)

	if w, ok := any(item.val).(interface{ Wait() error }); ok {
		wait = make(chan error, 1)
		go func() {
			wait <- w.Wait()
			close(wait)
		}()
	}

	for {
		select {
		case ctx := <-item.refs:
			if ctx.Err() != nil {
				break
			}
			refCount++
			timeout = nil
			go func() { done <- <-ctx.Done() }()
		case <-done:
			refCount--
			if refCount == 0 {
				timeout = time.After(p.timeout)
			}
		case <-wait:
			p.mu.Lock()
			delete(p.items, key)
			close(item.done)
			p.mu.Unlock()
			return
		case <-timeout:
			p.mu.Lock()
			delete(p.items, key)
			close(item.done)
			if c, ok := any(item.val).(io.Closer); ok {
				c.Close()
			}
			p.mu.Unlock()
			return
		}
	}
}
