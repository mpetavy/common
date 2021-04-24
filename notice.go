package common

import "sync"

type Notice struct {
	sync.Mutex

	isSet     bool
	lastIsSet bool
	chs       []chan struct{}
}

func NewNotice() *Notice {
	return &Notice{isSet: false, chs: make([]chan struct{}, 0)}
}

func (this *Notice) NewChannel() chan struct{} {
	ch := make(chan struct{})

	this.chs = append(this.chs, ch)

	return ch
}

func (this *Notice) IsSet() bool {
	this.Lock()
	defer this.Unlock()

	if this.lastIsSet != this.isSet {
		this.lastIsSet = this.isSet

		DebugFunc(this.isSet)
	}

	return this.isSet
}

func (this *Notice) Set() bool {
	this.Lock()
	defer this.Unlock()

	if !this.isSet {
		DebugFunc()

		this.isSet = true

		this.chs = make([]chan struct{}, 0)

		return true
	}

	return false
}

func (this *Notice) Unset() bool {
	this.Lock()
	defer this.Unlock()

	if this.isSet {
		DebugFunc()

		this.isSet = false

		for i := 0; i < len(this.chs); i++ {
			close(this.chs[i])
		}

		return true
	}

	return false
}
