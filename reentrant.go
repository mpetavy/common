package common

import (
	"fmt"
	"sync"
)

var ErrReentrantNotOwner = fmt.Errorf("reentrantmutex: unlock called by non-owner")

type ReentrantMutex struct {
	PreventIfSame bool
	mu            sync.Mutex
	cond          *sync.Cond
	owner         uint64 // goroutine id
	count         uint64
}

func NewReentrantMutex(preventIfSame bool) *ReentrantMutex {
	m := &ReentrantMutex{
		PreventIfSame: preventIfSame,
	}
	m.cond = sync.NewCond(&m.mu)

	return m
}

// TryLock tries to acquire the lock.
// Returns true if the lock was acquired or re-entered, false if re-entry is not allowed.
func (m *ReentrantMutex) TryLock() bool {
	gid := GoRoutineId()

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cond == nil {
		m.cond = sync.NewCond(&m.mu)
	}

	if m.owner == gid {
		if m.PreventIfSame {
			return false
		}
		m.count++
		return true
	}

	if m.count == 0 {
		m.owner = gid
		m.count = 1
		return true
	}
	return false
}

// Lock blocks until the lock is acquired or re-entered.
func (m *ReentrantMutex) Lock() {
	gid := GoRoutineId()

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cond == nil {
		m.cond = sync.NewCond(&m.mu)
	}

	if m.owner == gid {
		if m.PreventIfSame {
			return
		}
		m.count++
		return
	}

	for m.count != 0 {
		m.cond.Wait()
	}

	m.owner = gid
	m.count = 1
}

func (m *ReentrantMutex) Unlock() {
	DebugError(m.TryUnlock())
}

// Unlock releases the lock once.
func (m *ReentrantMutex) TryUnlock() error {
	gid := GoRoutineId()

	m.mu.Lock()
	defer func() {
		m.cond.Broadcast()
		m.mu.Unlock()
	}()

	if m.cond == nil {
		m.cond = sync.NewCond(&m.mu)
	}

	if m.owner != gid {
		return ErrReentrantNotOwner
	}

	m.count--
	if m.count == 0 {
		m.owner = 0
	}
	return nil
}

// UnlockNow fully releases the lock regardless of the reentrancy count.
func (m *ReentrantMutex) UnlockNow() error {
	gid := GoRoutineId()

	m.mu.Lock()
	defer func() {
		m.cond.Broadcast()
		m.mu.Unlock()
	}()

	if m.cond == nil {
		m.cond = sync.NewCond(&m.mu)
	}

	if m.owner != gid {
		return ErrReentrantNotOwner
	}

	m.count = 0
	m.owner = 0
	return nil
}
