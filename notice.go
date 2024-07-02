package common

import (
	"sync"
	"time"
)

type Notice struct {
	isSet bool
	mu    sync.Mutex
	ch    chan struct{}
}

func NewNotice() *Notice {
	return &Notice{
		isSet: true,
		mu:    sync.Mutex{},
		ch:    make(chan struct{}),
	}
}

func (this *Notice) Channel() chan struct{} {
	this.mu.Lock()
	defer this.mu.Unlock()

	if this.ch == nil {
		this.ch = make(chan struct{})

		time.AfterFunc(time.Millisecond, func() {
			close(this.ch)
		})
	}

	return this.ch
}

func (this *Notice) IsSet() bool {
	this.mu.Lock()
	defer this.mu.Unlock()

	return this.isSet
}

func (this *Notice) Set() bool {
	this.mu.Lock()
	defer this.mu.Unlock()

	if !this.isSet {
		this.isSet = true
		this.ch = make(chan struct{})

		return true
	}

	return false
}

func (this *Notice) Unset() bool {
	this.mu.Lock()
	defer this.mu.Unlock()

	if this.isSet {
		this.isSet = false

		close(this.ch)
		this.ch = nil

		return true
	}

	return false
}
