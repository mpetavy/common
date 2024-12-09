package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestKeyValues(t *testing.T) {
	kvl := KeyValues{}

	kvl.Put("a", "1")
	require.Equal(t, 1, len(kvl))
	kvl.Put("b", "2")
	require.Equal(t, 2, len(kvl))
	kvl.Put("c", "3")
	require.Equal(t, 3, len(kvl))

	a, _ := kvl.Get("a")
	b, _ := kvl.Get("b")
	c, _ := kvl.Get("c")

	keys := kvl.Keys()
	require.Equal(t, "a", keys[0])
	require.Equal(t, "b", keys[1])
	require.Equal(t, "c", keys[2])

	require.Equal(t, "1", a)
	require.Equal(t, "2", b)
	require.Equal(t, "3", c)

	x, err := kvl.Get("x")
	require.Equal(t, "", x)
	require.NotEqual(t, nil, err)

	kvl.Put("b", "99")
	b, _ = kvl.Get("b")
	require.Equal(t, "99", b)
	require.Equal(t, 3, len(kvl))

	c, _ = kvl.Remove("c")
	require.Equal(t, "3", c)
	require.Equal(t, 2, len(kvl))
	b, _ = kvl.Remove("b")
	require.Equal(t, "99", b)
	require.Equal(t, 1, len(kvl))
	a, _ = kvl.Remove("a")
	require.Equal(t, "1", a)
	require.Equal(t, 0, len(kvl))

	kvl.Put("a", "1")
	kvl.Put("b", "2")
	kvl.Put("c", "3")

	kvl.Clear()

	require.Equal(t, 0, len(kvl))
}

func TestDuplicates(t *testing.T) {
	kvl := KeyValues{}

	kvl.Put("a", "1")
	require.Equal(t, 1, len(kvl))
	kvl.Put("a", "1")
	require.Equal(t, 1, len(kvl))
	kvl.Add("a", "1")
	require.Equal(t, 2, len(kvl))
}

func TestContains(t *testing.T) {
	kvl := KeyValues{}

	kvl.Put("a", "1")

	require.True(t, kvl.Contains("a"))
	require.False(t, kvl.Contains("x"))
}
