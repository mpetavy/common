package common

import (
	"sync"
	"sync/atomic"
)

// ReentrantMutex is a Mutex which shall prevent a function to enter if it is already taken by an other GO routine
// If the same GO routine wants to enter then this can be choosen by setting EnterIfSame
// For example the logging functionality want to prevent this but any

type GoRoutineMutex struct {
	EnterIfSame bool
	mu          sync.Mutex
	current     atomic.Uint64
}

func (m *GoRoutineMutex) TryLock() bool {
	id := GoRoutineId()

	for {
		// if there is no current holder of the lock, then the current GO routine will be and code can enter
		if m.current.CompareAndSwap(0, id) {
			return true
		}

		// if the current holder is already the same GO routine then do not enter the following code (reentrant)
		if m.current.CompareAndSwap(id, id) {
			return m.EnterIfSame
		}

		// we loop again to simulate a lock ...
	}
}

func (m *GoRoutineMutex) Unlock() {
	m.current.Store(0)
}
