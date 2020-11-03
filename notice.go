package common

import "sync"

type Notice struct {
	sync.Mutex

	b   bool
	chs []chan struct{}
}

func NewNotice() *Notice {
	return &Notice{b: false, chs: make([]chan struct{}, 0)}
}

func (this *Notice) NewChannel() chan struct{} {
	ch := make(chan struct{})

	this.chs = append(this.chs, ch)

	return ch
}

func (this *Notice) IsSet() bool {
	this.Lock()
	defer this.Unlock()

	return this.b
}

func (this *Notice) Set() bool {
	this.Lock()
	defer this.Unlock()

	if !this.b {
		this.b = true

		this.chs = make([]chan struct{}, 0)

		return true
	}

	return false
}

func (this *Notice) Unset() bool {
	this.Lock()
	defer this.Unlock()

	if this.b {
		this.b = false

		for i := 0; i < len(this.chs); i++ {
			close(this.chs[i])
		}

		return true
	}

	return false
}
