package common

import "sync"

type Sync[T any] struct {
	mu    sync.RWMutex
	isSet bool
	value T
}

func (cv *Sync[T]) IsSet() bool {
	cv.mu.RLock()
	defer cv.mu.RUnlock()

	return cv.isSet
}

func (cv *Sync[T]) Get() T {
	cv.mu.RLock()
	defer cv.mu.RUnlock()

	return cv.value
}

func (cv *Sync[T]) Set(value T) {
	cv.mu.Lock()
	defer cv.mu.Unlock()

	cv.isSet = true
	cv.value = value
}

func (cv *Sync[T]) Run(fn func(T)) {
	cv.mu.Lock()
	defer cv.mu.Unlock()

	fn(cv.value)
}
