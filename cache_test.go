package common

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	capacity := 10

	cache := NewCache[int, int](capacity)

	for i := range capacity {
		require.NoError(t, cache.Put(i, i))
	}

	// Expect filled up cache
	require.Equal(t, capacity, cache.Len())

	// Add new Value n and expect to be at first. Also expect that oldest values 0 is deleted
	n := capacity
	require.NoError(t, cache.Put(n, n))
	require.Equal(t, cache.KeyAt(0), n)
	require.Equal(t, -1, cache.KeyIndex(0))

	// Remove n from cache and expect cache len is capacit-1
	require.NoError(t, cache.Remove(n))
	require.Equal(t, -1, cache.KeyIndex(n))
	require.Equal(t, capacity-1, cache.Len())

	// cache put 3 and expect to be at pposition 0
	require.False(t, cache.KeyIndex(3) == 0)
	require.NoError(t, cache.Put(3, 3))
	require.True(t, cache.KeyIndex(3) == 0)

	// cache get 5 and expect tp be at pposition 0
	require.False(t, cache.KeyIndex(5) == 0)
	_, err := cache.Get(5)
	require.NoError(t, err)
	require.True(t, cache.KeyIndex(5) == 0)

	require.Equal(t, capacity-1, cache.Len())

	// expect that 99 is not in cache
	_, err = cache.Get(99)
	require.Error(t, err)
}

func TestCachePerformance(t *testing.T) {
	cache := NewCache[int, int](100000)
	for i := range cache.capacity {
		require.NoError(t, cache.Put(i, i))
	}
	start := time.Now()
	for range cache.capacity {
		v := Rnd(cache.capacity)
		_, err := cache.Get(v)
		require.NoError(t, err)
	}
	fmt.Printf("%v\n", time.Since(start))
}
