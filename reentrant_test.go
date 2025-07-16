package common

import (
	"github.com/stretchr/testify/require"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestReentrantSimple(t *testing.T) {
	rm := ReentrantMutex{}

	rm.Lock()
	rm.Lock()
	rm.Lock()

	require.Equal(t, rm.count, uint64(3))

	rm.Unlock()

	require.Equal(t, rm.count, uint64(2))

	require.NoError(t, rm.UnlockNow())

	require.Equal(t, rm.count, uint64(0))
	require.Equal(t, rm.owner.Load(), int64(0))
}

func TestReentrantBlocking(t *testing.T) {
	rm := ReentrantMutex{}

	start := time.Now()
	wg := sync.WaitGroup{}
	i := atomic.Int32{}

	rm.Lock()
	i.Store(1)

	wg.Add(1)
	go func() {
		defer wg.Done()

		rm.Lock()
		i.CompareAndSwap(1, 2)
	}()

	time.Sleep(time.Millisecond * 100)

	rm.Unlock()

	wg.Wait()

	require.LessOrEqual(t, int64(100), time.Since(start).Milliseconds())

	require.True(t, i.Load() == 2)
}

func TestReentrantPreventIfSame(t *testing.T) {
	rm := NewReentrantMutex(true)

	require.True(t, rm.TryLock())
	require.False(t, rm.TryLock())
	require.False(t, rm.TryLock())

	require.Equal(t, rm.count, uint64(1))

	rm.Unlock()

	require.Equal(t, rm.count, uint64(0))

	rm = NewReentrantMutex(false)

	require.True(t, rm.TryLock())
	require.True(t, rm.TryLock())
	require.True(t, rm.TryLock())

	require.Equal(t, rm.count, uint64(3))

	for c := range 3 {
		rm.Unlock()

		require.Equal(t, rm.count, uint64(2-c))
	}

	require.Equal(t, rm.count, uint64(0))
	require.Equal(t, rm.owner.Load(), int64(0))
}

func TestReentrantUnlockNow(t *testing.T) {
	rm := ReentrantMutex{}

	require.True(t, rm.TryLock())
	require.True(t, rm.TryLock())
	require.True(t, rm.TryLock())

	require.Equal(t, rm.count, uint64(3))

	require.NoError(t, rm.UnlockNow())

	require.Equal(t, rm.count, uint64(0))
	require.Equal(t, rm.owner.Load(), int64(0))
}
