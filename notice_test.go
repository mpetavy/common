package common

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestNoticeChannel(t *testing.T) {
	n := NewNotice()
	n.Set()
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	c := 0

	for i := 0; i < 10; i++ {
		c++

		wg.Add(1)
		go func() {
			defer wg.Done()

			<-n.NewChannel()
			mu.Lock()
			defer mu.Unlock()

			c--
		}()
	}

	time.Sleep(time.Second)

	n.Unset()

	wg.Wait()

	assert.Equal(t, 0, c)
}
