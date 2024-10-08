package common

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGoRoutineVars(t *testing.T) {
	ids := make(map[uint64]string, 0)

	quit := make(chan struct{})

	for i := range 3 {
		go func(i int) {
			id := GoRoutineId()
			value := fmt.Sprintf("%d", id)

			ids[id] = value

			GoRoutineVars.Set("value", value)

			<-quit
		}(i)
	}

	time.Sleep(time.Second)

	for k, v := range ids {
		value := GoRoutineVars.GetById(k)["value"]

		require.Equal(t, v, value)
	}

	close(quit)

	time.Sleep(time.Second)

	for k := range ids {
		value := GoRoutineVars.GetById(k)["value"]

		require.Nil(t, value)
	}
}
