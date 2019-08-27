package common

import "sync"

type Sign struct {
	sync.Mutex

	isSet int
}

func (this *Sign) Set() bool {
	this.Lock()
	defer this.Unlock()

	if this.isSet == 0 {
		this.isSet = 1

		return true
	}

	return false
}

func (this *Sign) Unset() bool {
	this.Lock()
	defer this.Unlock()

	if this.isSet == 1 {
		this.isSet = 0

		return true
	}

	return false
}

func (this *Sign) Inc() int {
	this.Lock()
	defer this.Unlock()

	this.isSet++

	return this.isSet
}

func (this *Sign) Dec() int {
	this.Lock()
	defer this.Unlock()

	this.isSet--

	return this.isSet
}

func (this *Sign) Reset() {
	this.Lock()
	defer this.Unlock()

	this.isSet = 0
}

func (this *Sign) ResetWithoutLock() {
	this.isSet = 0
}

func (this *Sign) IncAndReached(v int) bool {
	this.Lock()

	this.isSet++

	if this.isSet == v {
		return true
	}

	this.Unlock()

	return false
}
