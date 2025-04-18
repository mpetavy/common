package common

import (
	"sync"
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
	}
}

func (this *Notice) Channel() chan struct{} {
	this.mu.Lock()
	defer this.mu.Unlock()

	if this.ch == nil {
		this.ch = make(chan struct{})
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

		return true
	}

	return false
}

func (this *Notice) Unset() bool {
	this.mu.Lock()
	defer this.mu.Unlock()

	if this.isSet {
		this.isSet = false

		if this.ch != nil {
			close(this.ch)
			this.ch = nil
		}

		return true
	}

	return false
}
