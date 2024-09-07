package common

type Sync[T any] struct {
	ReentrantMutex
	isSet bool
	Ref   T
}

func NewSync[T any]() *Sync[T] {
	return &Sync[T]{}
}

func NewSyncOf[T any](t T) *Sync[T] {
	return &Sync[T]{
		Ref:   t,
		isSet: true,
	}
}

func (sync *Sync[T]) IsSet() bool {
	sync.Lock()
	defer sync.Unlock()

	return sync.isSet
}

func (sync *Sync[T]) Get() T {
	sync.Lock()
	defer sync.Unlock()

	clone := sync.Ref

	return clone
}

func (sync *Sync[T]) Set(value T) {
	sync.Lock()
	defer sync.Unlock()

	sync.isSet = true
	sync.Ref = value
}

func (sync *Sync[T]) Run(fn func(T)) {
	sync.Lock()
	defer sync.Unlock()

	fn(sync.Ref)
}
