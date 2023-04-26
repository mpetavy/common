package common

import (
	"sync"
)

type ReentrantMutex struct {
	sync.Locker

	mu      sync.Mutex
	current uint64
	count   int
}

func NewRentrantMutex() ReentrantMutex {
	return ReentrantMutex{
		mu: sync.Mutex{},
	}
}

func (rm *ReentrantMutex) Lock() {
	id := GoRoutineId()

	for {
		rm.mu.Lock()

		if rm.current != 0 && rm.current != id {
			rm.mu.Unlock()

			continue
		}

		if rm.current == 0 {
			rm.current = id
		}

		rm.count++

		rm.mu.Unlock()

		break
	}
}

func (rm *ReentrantMutex) UnlockNow() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.current = 0
	rm.count = 0
}

func (rm *ReentrantMutex) Unlock() {
	id := GoRoutineId()

	for {
		rm.mu.Lock()

		if rm.current != id {
			rm.mu.Unlock()

			continue
		}

		if rm.count > 0 {
			rm.count--
		}

		if rm.count == 0 {
			rm.current = 0
		}

		rm.mu.Unlock()

		break
	}
}
