package common

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestNoticeChannel(t *testing.T) {
	n := NewNotice(true)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	c := 0

	ch := n.NewChannel()
	defer n.RemoveChannel(ch)

	for i := 0; i < 10; i++ {
		c++

		wg.Add(1)
		go func() {
			defer UnregisterGoRoutine(RegisterGoRoutine(1))

			defer wg.Done()

			<-ch

			mu.Lock()
			defer mu.Unlock()

			c--
		}()
	}

	Sleep(time.Second)

	n.Unset()

	wg.Wait()

	assert.Equal(t, 0, c)
}
