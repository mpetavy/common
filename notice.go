package common

import "sync"

type Notice struct {
	sync.Mutex

	isSet int
	ch    chan struct{}
}

func NewNotice() *Notice {
	return &Notice{isSet: 0, ch: make(chan struct{})}
}

func (this *Notice) Channel() chan struct{} {
	return this.ch
}

func (this *Notice) IsSet() bool {
	this.Lock()
	defer this.Unlock()

	return this.isSet > 0
}

func (this *Notice) Set() bool {
	this.Lock()
	defer this.Unlock()

	if this.isSet == 0 {
		this.isSet = 1

		close(this.ch)

		return true
	}

	return false
}

func (this *Notice) Unset() bool {
	this.Lock()
	defer this.Unlock()

	if this.isSet == 1 {
		this.isSet = 0

		this.ch = make(chan struct{})

		return true
	}

	return false
}

func (this *Notice) Inc() int {
	this.Lock()
	defer this.Unlock()

	this.isSet++

	return this.isSet
}

func (this *Notice) Dec() int {
	this.Lock()
	defer this.Unlock()

	this.isSet--

	return this.isSet
}

func (this *Notice) Reset() {
	this.Lock()
	defer this.Unlock()

	this.isSet = 0
}

func (this *Notice) ResetWithoutLock() {
	this.isSet = 0
}

func (this *Notice) IncAndReached(v int) bool {
	this.Lock()

	this.isSet++

	if this.isSet == v {
		return true
	}

	this.Unlock()

	return false
}
