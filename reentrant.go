package common

import (
	"fmt"
	"sync"
)

// ReentrantMutex is a mutex that can be locked multiple times by the same goroutine.
type ReentrantMutex struct {
	EnterIfSame bool
	owner       uint64
	count       uint64
	mu          sync.Mutex
	muUpdate    sync.Mutex
}

var ErrReentrantNotOwner = fmt.Errorf("reentrantmutex: unlock called by non-owner")

func NewReentrantMutex() *ReentrantMutex {
	return &ReentrantMutex{}
}

func (m *ReentrantMutex) Lock() {
	gid := GoRoutineId()

	alreadyOwned := false

	m.muUpdate.Lock()

	if m.owner == gid {
		m.count++

		alreadyOwned = true
	}

	m.muUpdate.Unlock()

	if alreadyOwned {
		return
	}

	m.mu.Lock()

	m.owner = gid
	m.count = 1
}

func (m *ReentrantMutex) TryLock() bool {
	gid := GoRoutineId()

	alreadyOwned := false

	m.muUpdate.Lock()

	if m.owner == gid {
		alreadyOwned = true

		if m.EnterIfSame {
			m.count++
		}
	}

	m.muUpdate.Unlock()

	if alreadyOwned {
		return m.EnterIfSame
	}

	m.mu.Lock()

	m.owner = gid
	m.count = 1

	return true
}

func (m *ReentrantMutex) Unlock() {
	gid := GoRoutineId()

	m.muUpdate.Lock()

	if m.owner != gid {
		m.muUpdate.Unlock()

		Panic(ErrReentrantNotOwner)
	}

	doUnlock := false

	m.count--
	if m.count == 0 {
		doUnlock = true

		m.owner = 0
	}

	m.muUpdate.Unlock()

	if doUnlock {
		m.mu.Unlock()
	}
}

func (m *ReentrantMutex) UnlockNow() {
	gid := GoRoutineId()

	m.muUpdate.Lock()

	if m.owner != gid {
		m.muUpdate.Unlock()

		Panic(ErrReentrantNotOwner)
	}

	doUnlock := m.count > 0

	m.owner = 0
	m.count = 0

	m.muUpdate.Unlock()

	if doUnlock {
		m.mu.Unlock()
	}
}
