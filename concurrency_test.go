package common

import (
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"
)

var count atomic.Int64

func task(backgroundTask *BackgroundTask) {
	for backgroundTask.IsAlive() {
		count.Store(count.Load() + 1)

		Sleep(time.Millisecond * 10)
	}
}

func TestBackgroundTask(t *testing.T) {
	bt := NewBackgroundTask(task)

	bt.Start()

	Sleep(time.Millisecond * 100)

	bt.Stop(true)

	lastCount := count.Load()

	Sleep(time.Millisecond * 200)

	require.Equal(t, lastCount, count.Load())

	bt.Start()

	Sleep(time.Millisecond * 200)

	require.True(t, count.Load() > lastCount)

	bt.Stop(true)
}
