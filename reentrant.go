package common

import (
	"fmt"
	"sync"
	"sync/atomic"
)

var ErrReentrantNotOwner = fmt.Errorf("ReentrantMutex: unlock called by non-owner")

type ReentrantMutex struct {
	PreventIfSame bool
	owner         atomic.Int64
	count         uint64
	mu            sync.Mutex
}

func NewReentrantMutex(preventIfSame bool) *ReentrantMutex {
	m := &ReentrantMutex{
		PreventIfSame: preventIfSame,
	}

	return m
}

func (rm *ReentrantMutex) Lock() {
	id := int64(GoRoutineId())

	if rm.owner.CompareAndSwap(id, id) {
		rm.count++

		return
	}

	rm.mu.Lock()

	rm.owner.Store(id)
	rm.count = 1
}

func (rm *ReentrantMutex) TryLock() bool {
	id := int64(GoRoutineId())

	if rm.owner.CompareAndSwap(id, id) {
		if rm.PreventIfSame {
			return false
		}

		rm.count++

		return true
	}

	rm.mu.Lock()

	rm.owner.Store(id)
	rm.count = 1

	return true
}

func (rm *ReentrantMutex) Unlock() {
	id := int64(GoRoutineId())

	if rm.owner.CompareAndSwap(id, id) {
		rm.count--

		if rm.count == 0 {
			rm.owner.Store(0)
			rm.mu.Unlock()
		}
	}
}

func (rm *ReentrantMutex) UnlockNow() error {
	id := int64(GoRoutineId())

	if rm.owner.CompareAndSwap(id, 0) {
		rm.count = 0

		rm.mu.Unlock()

		return nil
	}

	return ErrReentrantNotOwner
}
