package common

import (
	"sync/atomic"
)

// ReentrantMutex is a Mutex which shall prevent a function to enter if it is already taken by an other GO routine
type ReentrantMutex struct {
	id           atomic.Uint64
	count        atomic.Uint64
	loopNotifier LoopNotifier
}

func NewReentrantMutex() *ReentrantMutex {
	return &ReentrantMutex{}
}

func (m *ReentrantMutex) TryLock() bool {
	id := GoRoutineId()

	if m.id.CompareAndSwap(0, id) || m.id.CompareAndSwap(id, id) {
		m.count.Store(m.count.Load() + 1)

		return true
	}

	return false
}

func (m *ReentrantMutex) Lock() {
	m.loopNotifier.Reset()

	for !m.TryLock() {
		m.loopNotifier.Notify()
	}
}

func (m *ReentrantMutex) Unlock() {
	id := GoRoutineId()

	if m.id.CompareAndSwap(id, id) {
		m.count.Store(Max(0, m.count.Load()-1))

		if m.count.Load() == 0 {
			m.id.Store(0)
		}
	}
}

func (m *ReentrantMutex) UnlockNow() {
	id := GoRoutineId()

	if m.id.CompareAndSwap(id, id) {
		m.count.Store(0)
		m.id.Store(0)
	}
}
