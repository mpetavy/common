package common

import (
	"sync"
)

// FIXME add to notice
// ObjectMonitor mimics Java's wait/notify/notifyAll methods, and adds channel-based wait support.
type ObjectMonitor struct {
	mu      sync.Mutex
	cond    *sync.Cond
	waiters []chan struct{}
}

// NewObjectMonitor constructs a new ObjectMonitor.
func NewObjectMonitor() *ObjectMonitor {
	om := &ObjectMonitor{}
	om.cond = sync.NewCond(&om.mu)
	return om
}

// Wait blocks until notified (classic sync.Cond style).
func (om *ObjectMonitor) Wait() {
	om.mu.Lock()
	om.cond.Wait()
	om.mu.Unlock()
}

// Notify wakes up one waiting goroutine (classic sync.Cond style).
func (om *ObjectMonitor) Notify() {
	om.mu.Lock()
	om.cond.Signal()
	// Channel-based notify
	if len(om.waiters) > 0 {
		ch := om.waiters[0]
		om.waiters = om.waiters[1:]
		close(ch)
	}
	om.mu.Unlock()
}

// NotifyAll wakes up all waiting goroutines (classic sync.Cond style).
func (om *ObjectMonitor) NotifyAll() {
	om.mu.Lock()
	om.cond.Broadcast()
	// Channel-based notify all
	for _, ch := range om.waiters {
		close(ch)
	}
	om.waiters = nil
	om.mu.Unlock()
}

// WaitChan returns a channel that will be closed when notified.
// The caller can select or receive from this channel to wait.
func (om *ObjectMonitor) WaitChan() <-chan struct{} {
	ch := make(chan struct{})
	om.mu.Lock()
	om.waiters = append(om.waiters, ch)
	om.mu.Unlock()
	return ch
}
