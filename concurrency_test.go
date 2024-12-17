package common

import (
	"flag"
	"github.com/stretchr/testify/require"
	"strconv"
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

func TestAlignedTicker(t *testing.T) {
	stopAt := time.Now().Add(time.Second * 5)

	at := NewAlignedTicker(time.Second * 2)

	for time.Now().Before(stopAt) {
		sleep := at.SleepUntilNextTicker()

		time.Sleep(sleep)

		require.True(t, time.Now().Second()%int(at.SleepTime.Seconds()) == 0)
	}
}

func TestConcurrentLimit(t *testing.T) {
	orgTimeout := *FLagConcurrentTimeout

	start := time.Now()
	count := 0

	// can we register "with channel"
	err := flag.Set(FlagNameConcurrentTimeout, "500")
	require.NoError(t, err)

	for range *FlagConcurrentLimit {
		b := RegisterConcurrentLimit()

		count++

		require.True(t, b)
	}

	require.True(t, time.Now().Sub(start) < MillisecondToDuration(*FlagConcurrentLimit))

	// now we expect a timeout ...

	start = time.Now()
	b := RegisterConcurrentLimit()
	require.False(t, b)
	require.True(t, time.Now().Sub(start) >= MillisecondToDuration(*FlagConcurrentLimit))

	// and here also a timeout ...

	start = time.Now()
	b = RegisterConcurrentLimit()
	require.False(t, b)
	require.True(t, time.Now().Sub(start) >= MillisecondToDuration(*FlagConcurrentLimit))

	// ok now we reset the channel

	for range count {
		UnregisterConcurrentLimit(true)
	}

	// now we should be able to register "with channel" normal agaon
	start = time.Now()
	count = 0

	err = flag.Set(FlagNameConcurrentTimeout, "1000")
	require.NoError(t, err)

	for range *FlagConcurrentLimit {
		b := RegisterConcurrentLimit()

		count++

		require.True(t, b)
	}

	require.True(t, time.Now().Sub(start) < MillisecondToDuration(*FlagConcurrentLimit))

	// reset to default

	err = flag.Set(FlagNameConcurrentTimeout, strconv.Itoa(orgTimeout))
	require.NoError(t, err)
}
