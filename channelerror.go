package common

import (
	"sync"
)

type ChannelError struct {
	m sync.Mutex
	l []error
}

func (c *ChannelError) Add(err error) {
	c.m.Lock()
	c.l = append(c.l, err)
	c.m.Unlock()
}

func (c *ChannelError) Get() error {
	c.m.Lock()
	defer c.m.Unlock()

	if len(c.l) > 0 {
		return c.l[0]
	} else {
		return nil
	}
}

func (c *ChannelError) GetAll() []error {
	return c.l
}

func (c *ChannelError) Exists() bool {
	c.m.Lock()
	defer c.m.Unlock()

	return len(c.l) > 0
}
