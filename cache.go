package common

import (
	"fmt"
	"slices"
	"sync"
)

type Cache[K comparable, V any] struct {
	capacity int
	keys     []K
	values   []V
	mu       sync.RWMutex
}

type ErrNotFound[T any] struct {
	What T
}

func (e *ErrNotFound[T]) Error() string {
	return fmt.Sprintf("Not found: %v", e.What)
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	return &Cache[K, V]{
		capacity: capacity,
		keys:     make([]K, 0),
		values:   make([]V, 0),
	}
}

func (cache *Cache[K, V]) Len() int {
	return len(cache.keys)
}

func (cache *Cache[K, V]) PutFunc(key K, fn func() (V, error)) error {
	cache.mu.Lock()
	defer func() {
		cache.mu.Unlock()
	}()

	p := slices.Index(cache.keys, key)
	if p != -1 {
		cache.keys = SliceMove(cache.keys, p, 0)
		cache.values = SliceMove(cache.values, p, 0)

		return nil
	}

	if len(cache.keys) >= cache.capacity {
		cache.keys = slices.Delete(cache.keys, cache.capacity-1, len(cache.keys))
		cache.values = slices.Delete(cache.values, cache.capacity-1, len(cache.values))
	}

	value, err := fn()
	if Error(err) {
		return err
	}

	cache.keys = slices.Insert(cache.keys, 0, key)
	cache.values = slices.Insert(cache.values, 0, value)

	return nil
}

func (cache *Cache[K, V]) Put(key K, value V) error {
	return cache.PutFunc(key, func() (V, error) {
		return value, nil
	})
}

func (cache *Cache[K, V]) Get(key K) (V, error) {
	cache.mu.Lock()
	defer func() {
		cache.mu.Unlock()
	}()

	p := slices.Index(cache.keys, key)
	if p != -1 {
		cache.keys = SliceMove(cache.keys, p, 0)
		cache.values = SliceMove(cache.values, p, 0)

		return cache.values[0], nil
	}

	value := new(V)

	return *value, &ErrNotFound[K]{What: key}
}

func (cache *Cache[K, V]) Remove(key K) error {
	cache.mu.Lock()
	defer func() {
		cache.mu.Unlock()
	}()

	p := slices.Index(cache.keys, key)
	if p == -1 {
		return &ErrNotFound[K]{What: key}
	}

	cache.keys = slices.Delete(cache.keys, p, p+1)
	cache.values = slices.Delete(cache.values, p, p+1)

	return nil
}
