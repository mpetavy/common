package common

import "sync"

type Sync[T any] struct {
	mu    sync.RWMutex
	isSet bool
	ref   *T
}

func NewSync[T any](t *T) *Sync[T] {
	return &Sync[T]{
		ref: t,
	}
}

func (sync *Sync[T]) IsSet() bool {
	sync.mu.RLock()
	defer sync.mu.RUnlock()

	return sync.isSet
}

func (sync *Sync[T]) Get() *T {
	sync.mu.RLock()
	defer sync.mu.RUnlock()

	return sync.ref
}

func (sync *Sync[T]) Set(value *T) {
	sync.mu.Lock()
	defer sync.mu.Unlock()

	sync.isSet = true
	sync.ref = value
}

func (sync *Sync[T]) Run(fn func(*T)) {
	sync.mu.Lock()
	defer sync.mu.Unlock()

	fn(sync.ref)
}
