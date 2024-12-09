package common

import (
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestNoticeChannel(t *testing.T) {
	n := NewNotice()
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	c := 0

	for i := 0; i < 10; i++ {
		c++

		wg.Add(1)
		go func() {
			defer UnregisterGoRoutine(RegisterGoRoutine(1))

			defer wg.Done()

			<-n.Channel()

			mu.Lock()
			defer mu.Unlock()

			c--
		}()
	}

	Sleep(time.Second)

	n.Unset()

	wg.Wait()

	require.Equal(t, 0, c)
}
