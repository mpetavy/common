package common

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestGoRoutineVars(t *testing.T) {
	mu := sync.Mutex{}
	ids := make(map[uint64]string, 0)

	quit := make(chan struct{})

	for i := range 3 {
		go func(i int) {
			id := GoRoutineId()
			value := fmt.Sprintf("%d", id)

			mu.Lock()
			ids[id] = value
			mu.Unlock()

			GoRoutineVars.Get().Set("value", value)

			<-quit
		}(i)
	}

	time.Sleep(time.Second)

	for k, v := range ids {
		value, ok := GoRoutineVars.Get().GetById(k, "value")
		require.True(t, ok)

		require.Equal(t, v, value)
	}

	close(quit)

	time.Sleep(time.Second)

	for k := range ids {
		value, ok := GoRoutineVars.Get().GetById(k, "value")
		require.False(t, ok)

		require.Nil(t, value)
	}
}
