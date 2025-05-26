package common

import (
	"sync"
	"sync/atomic"
)

type Notice struct {
	isSet atomic.Bool
	mu    sync.Mutex
	chs   []chan struct{}
	fn    func(notice *Notice)
}

func NewNotice(isSet bool, fn func(notice *Notice)) *Notice {
	notice := &Notice{
		mu: sync.Mutex{},
		fn: fn,
	}
	notice.isSet.Store(isSet)

	return notice
}

func (notice *Notice) Channel() chan struct{} {
	notice.mu.Lock()
	defer notice.mu.Unlock()

	ch := make(chan struct{})

	notice.chs = append(notice.chs, ch)

	return ch
}

func (notice *Notice) IsSet() bool {
	return notice.isSet.Load()
}

func (notice *Notice) Set() bool {
	if notice.isSet.CompareAndSwap(false, true) {
		if notice.fn != nil {
			notice.fn(notice)
		}

		return true
	}

	return false
}

func (notice *Notice) Unset() bool {
	if notice.isSet.CompareAndSwap(true, false) {
		notice.mu.Lock()

		for i := 0; i < len(notice.chs); i++ {
			close(notice.chs[i])
		}

		notice.chs = nil

		notice.mu.Unlock()

		if notice.fn != nil {
			notice.fn(notice)
		}

		return true
	}

	return false
}

func (notice *Notice) Func() {
	if notice.fn != nil {
		notice.fn(notice)
	}
}
