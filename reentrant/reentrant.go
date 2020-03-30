package reentrant

import (
	"fmt"
	"sync"
)

type pass struct {
	c int
}

type Lock struct {
	mu sync.Mutex

	ch      chan interface{}
	current *pass
}

func New() Lock {
	return Lock{
		mu:      sync.Mutex{},
		ch:      make(chan interface{}),
		current: nil,
	}
}

func NewPass() *pass {
	return &pass{}
}

func (this *Lock) Lock(p *pass) {
	this.mu.Lock()

	if this.current == nil {
		p.c++

		this.ch = make(chan interface{})
		this.current = p

		this.mu.Unlock()

		return
	}

	if this.current == p {
		p.c++

		this.mu.Unlock()

		return
	}

	this.mu.Unlock()

	for {
		<-this.ch

		this.mu.Lock()

		if this.current == nil {
			p.c++

			this.ch = make(chan interface{})
			this.current = p

			this.mu.Unlock()

			return
		} else {
			this.mu.Unlock()
		}
	}
}

func (this *Lock) UnlockNow() {
	this.mu.Lock()
	defer this.mu.Unlock()

	this.current = nil

	if this.ch != nil {
		close(this.ch)

		this.ch = nil
	}
}

func (this *Lock) Unlock(p *pass) {
	for {
		this.mu.Lock()

		switch {
		case this.current == nil:
			this.current = nil

			if this.ch != nil {
				close(this.ch)

				this.ch = nil
			}

			this.mu.Unlock()

			return
		case this.current == p:
			this.current.c--

			if this.current.c == 0 {
				this.current = nil

				if this.ch != nil {
					close(this.ch)

					this.ch = nil
				}
			}

			this.mu.Unlock()

			return
		default:
			panic(fmt.Errorf("invalid pass"))
		}
	}
}

func (this *Lock) HasLock() bool {
	this.mu.Lock()
	defer this.mu.Unlock()

	return this.current != nil
}
