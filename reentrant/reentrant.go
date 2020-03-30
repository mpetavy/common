package reentrant

import (
	"fmt"
	"sync"
)

type pass struct {
	remux *reentrantmutex
	c     int
}

type reentrantmutex struct {
	mu      sync.Mutex
	current *pass
}

func NewRentrantMutex() reentrantmutex {
	return reentrantmutex{
		mu:      sync.Mutex{},
		current: nil,
	}
}

func (this *reentrantmutex) NewPass() *pass {
	return &pass{
		remux: this,
		c:     0,
	}
}

func (this *pass) Lock() {
	this.remux.mu.Lock()

	if this.remux.current == nil {
		this.c++

		this.remux.current = this

		this.remux.mu.Unlock()

		return
	}

	if this.remux.current == this {
		this.c++

		this.remux.mu.Unlock()

		return
	}

	this.remux.mu.Unlock()

	for {
		this.remux.mu.Lock()

		if this.remux.current == nil {
			this.c++

			this.remux.current = this

			this.remux.mu.Unlock()

			return
		} else {
			this.remux.mu.Unlock()
		}
	}
}

func (this *reentrantmutex) UnlockNow() {
	this.mu.Lock()
	defer this.mu.Unlock()

	this.current = nil
}

func (this *pass) Unlock() {
	for {
		this.remux.mu.Lock()

		switch {
		case this.remux.current == nil:
			this.remux.current = nil

			this.remux.mu.Unlock()

			return
		case this.remux.current == this:
			this.remux.current.c--

			if this.remux.current.c == 0 {
				this.remux.current = nil
			}

			this.remux.mu.Unlock()

			return
		default:
			panic(fmt.Errorf("invalid pass"))
		}
	}
}

func (this *reentrantmutex) HasLock() bool {
	this.mu.Lock()
	defer this.mu.Unlock()

	return this.current != nil
}
