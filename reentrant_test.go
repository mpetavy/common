package common

import (
	"github.com/stretchr/testify/require"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestReentrantSimple(t *testing.T) {
	rm := NewReentrantMutex()

	rm.Lock()
	rm.Lock()
	rm.Lock()

	require.Equal(t, rm.count, uint64(3))

	rm.Unlock()

	require.Equal(t, rm.count, uint64(2))

	rm.UnlockNow()

	require.Equal(t, rm.count, uint64(0))
	require.Equal(t, rm.owner, uint64(0))
}

func TestReentrantBlocking(t *testing.T) {
	rm := NewReentrantMutex()

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

func TestReentrantEnterIfSame(t *testing.T) {
	m := NewReentrantMutex()
	m.EnterIfSame = false

	require.True(t, m.TryLock())
	require.False(t, m.TryLock())
	require.False(t, m.TryLock())

	require.Equal(t, m.count, uint64(1))

	m.Unlock()

	require.Equal(t, m.count, uint64(0))

	m = NewReentrantMutex()
	m.EnterIfSame = true

	require.True(t, m.TryLock())
	require.True(t, m.TryLock())
	require.True(t, m.TryLock())

	require.Equal(t, m.count, uint64(3))

	for c := range 3 {
		m.Unlock()

		require.Equal(t, m.count, uint64(2-c))
	}

	require.Equal(t, m.count, uint64(0))
	require.Equal(t, m.owner, uint64(0))
}

func TestReentrantUnlockNow(t *testing.T) {
	m := NewReentrantMutex()
	m.EnterIfSame = true

	require.True(t, m.TryLock())
	require.True(t, m.TryLock())
	require.True(t, m.TryLock())

	require.Equal(t, m.count, uint64(3))

	m.UnlockNow()

	require.Equal(t, m.count, uint64(0))
	require.Equal(t, m.owner, uint64(0))
}
