package common

import "sync"

type Signal struct {
	sync.Mutex

	isSet int
	ch    chan struct{}
}

func NewSignal() *Signal {
	return &Signal{isSet: 0, ch: make(chan struct{})}
}

func (this *Signal) Channel() chan struct{} {
	return this.ch
}

func (this *Signal) IsSet() bool {
	this.Lock()
	defer this.Unlock()

	return this.isSet > 0
}

func (this *Signal) Set() bool {
	this.Lock()
	defer this.Unlock()

	if this.isSet == 0 {
		this.isSet = 1

		close(this.ch)

		return true
	}

	return false
}

func (this *Signal) Unset() bool {
	this.Lock()
	defer this.Unlock()

	if this.isSet == 1 {
		this.isSet = 0

		this.ch = make(chan struct{})

		return true
	}

	return false
}

func (this *Signal) Inc() int {
	this.Lock()
	defer this.Unlock()

	this.isSet++

	return this.isSet
}

func (this *Signal) Dec() int {
	this.Lock()
	defer this.Unlock()

	this.isSet--

	return this.isSet
}

func (this *Signal) Reset() {
	this.Lock()
	defer this.Unlock()

	this.isSet = 0
}

func (this *Signal) ResetWithoutLock() {
	this.isSet = 0
}

func (this *Signal) IncAndReached(v int) bool {
	this.Lock()

	this.isSet++

	if this.isSet == v {
		return true
	}

	this.Unlock()

	return false
}
