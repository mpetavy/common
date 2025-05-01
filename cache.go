package common

import (
	"slices"
	"sync"
)

type Cache[K comparable, V any] struct {
	capacity int
	keys     []K
	values   []V
	mu       sync.RWMutex
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

func (cache *Cache[K, V]) Put(key K, value V) {
	cache.mu.Lock()
	defer func() {
		cache.mu.Unlock()
	}()

	cache.keys = slices.Insert(cache.keys, 0, key)
	cache.values = slices.Insert(cache.values, 0, value)

	if len(cache.keys) > cache.capacity {
		cache.keys = slices.Delete(cache.keys, cache.capacity, cache.capacity+1)
		cache.values = slices.Delete(cache.values, cache.capacity, cache.capacity+1)
	}
}

func (cache *Cache[K, V]) Get(key K) V {
	cache.mu.Lock()
	defer func() {
		cache.mu.Unlock()
	}()

	p := slices.Index(cache.keys, key)

	if p == -1 {
		value := new(V)

		return *value
	}

	value := cache.values[p]

	cache.keys = slices.Delete(cache.keys, p, p+1)
	cache.values = slices.Delete(cache.values, p, p+1)

	cache.keys = slices.Insert(cache.keys, 0, key)
	cache.values = slices.Insert(cache.values, 0, value)

	return value
}

func (cache *Cache[K, V]) Remove(key K) {
	cache.mu.Lock()
	defer func() {
		cache.mu.Unlock()
	}()

	p := slices.Index(cache.keys, key)

	if p == -1 {
		return
	}

	cache.keys = slices.Delete(cache.keys, p, p+1)
	cache.values = slices.Delete(cache.values, p, p+1)
}