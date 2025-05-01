package common

import (
	"fmt"
)

// Cache is a thread-safe, LRU-based cache using a deque.
type Cache[K comparable, V any] struct {
	capacity int
	data     map[K]V
	order    *Deque[K]
	mu       ReentrantMutex // Ensure thread safety
}

type ErrNotFound[K comparable] struct {
	What K
}

func (e *ErrNotFound[K]) Error() string {
	return fmt.Sprintf("Not found: %v", e.What)
}

// NewCache creates a new cache with the specified capacity.
func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	return &Cache[K, V]{
		capacity: capacity,
		data:     make(map[K]V),
		order:    NewDeque[K](),
	}
}

// Len returns the current size of the cache.
func (c *Cache[K, V]) Len() int {
	c.mu.Lock() // Lock for reading
	defer c.mu.Unlock()
	return len(c.data)
}

// Put inserts a new value into the cache, or updates an existing one.
func (c *Cache[K, V]) PutFunc(key K, fn func() (V, error)) error {
	c.mu.Lock() // Lock for writing
	defer c.mu.Unlock()

	_, ok := c.data[key]
	if ok {
		c.order.PushFront(key)

		return nil
	}

	value, err := fn()
	if Error(err) {
		return err
	}

	if c.Len() >= c.capacity {
		// Remove least recently used item (from the back of the deque)
		leastUsedKey, _ := c.order.PopBack()
		delete(c.data, leastUsedKey)
	}

	// Insert or update the value
	c.data[key] = value
	// Move the key to the front of the deque (mark it as most recently used)
	c.order.PushFront(key)

	return nil
}

func (c *Cache[K, V]) Put(key K, value V) error {
	return c.PutFunc(key, func() (V, error) {
		return value, nil
	})
}

// Get retrieves a value from the cache and moves the key to the front (mark as recently used).
func (c *Cache[K, V]) Get(key K) (V, error) {
	c.mu.Lock() // Lock for writing (we move the key to the front, so it's a write operation)
	defer c.mu.Unlock()

	value, found := c.data[key]
	if !found {
		var zero V
		return zero, &ErrNotFound[K]{What: key}
	}

	// Move the key to the front of the deque (mark it as recently used)
	c.order.PushFront(key)
	return value, nil
}

// Remove removes a key-value pair from the cache.
func (c *Cache[K, V]) Remove(key K) error {
	c.mu.Lock() // Lock for writing
	defer c.mu.Unlock()

	_, found := c.data[key]
	if !found {
		return &ErrNotFound[K]{What: key}
	}

	// Remove from the deque and data
	_, err := c.order.PopFront() // or PopBack() depending on where the key is
	if Error(err) {
		return err
	}
	delete(c.data, key)
	return nil
}

func (c *Cache[K, V]) KeyIndex(key K) int {
	i := 0
	c.order.Iterate(func(current K) bool {
		if key == current {
			return false
		}

		i++

		return true
	})

	if i == len(c.data) {
		i = -1
	}

	return i
}

func (c *Cache[K, V]) KeyAt(index int) K {
	var found K
	i := 0
	c.order.Iterate(func(current K) bool {
		found = current
		if i == index {
			return false
		}

		i++

		return true
	})

	return found
}
