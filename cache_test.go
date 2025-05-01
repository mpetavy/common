package common

import (
	"github.com/stretchr/testify/require"
	"slices"
	"testing"
)

func TestCache(t *testing.T) {
	capacity := 10

	cache := NewCache[int, int](capacity)

	for i := range capacity + 1 {
		cache.Put(i, i)
	}

	require.Equal(t, capacity, cache.Len())

	n := capacity + 1

	cache.Put(n, n)

	require.Equal(t, cache.keys[0], n)
	require.Equal(t, -1, slices.Index(cache.keys, 0))

	cache.Remove(n)

	require.Equal(t, -1, slices.Index(cache.keys, n))
}