package common

import (
	"golang.org/x/exp/slices"
	"sync"
)

type Notice struct {
	isSet bool
	mu    sync.Mutex
	chs   []chan struct{}
}

func NewNotice(isSet bool) *Notice {
	return &Notice{
		isSet: isSet,
		chs:   make([]chan struct{}, 0),
	}
}

func (this *Notice) NewChannel() chan struct{} {
	this.mu.Lock()
	defer this.mu.Unlock()

	ch := make(chan struct{})

	this.chs = append(this.chs, ch)

	return ch
}

func (this *Notice) RemoveChannel(ch chan struct{}) {
	this.mu.Lock()
	defer this.mu.Unlock()

	p := slices.Index(this.chs, ch)
	if p != -1 {
		this.chs = slices.Delete(this.chs, p, p+1)
	}
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

		this.chs = make([]chan struct{}, 0)

		return true
	}

	return false
}

func (this *Notice) Unset() bool {
	this.mu.Lock()
	defer this.mu.Unlock()

	if this.isSet {
		this.isSet = false

		for len(this.chs) > 0 {
			close(this.chs[0])

			this.chs = this.chs[1:]
		}

		return true
	}

	return false
}
