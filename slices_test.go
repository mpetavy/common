package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSliceRemove(t *testing.T) {
	require.Equal(t, SliceRemove([]string{}, "a"), []string{})
	require.Equal(t, SliceRemove([]string{"a"}, "a"), []string{})
	require.Equal(t, SliceRemove([]string{"a", "b"}, "a"), []string{"b"})
	require.Equal(t, SliceRemove([]string{"a", "b"}, "b"), []string{"a"})
	require.Equal(t, SliceRemove([]string{"a", "b", "c"}, "a"), []string{"b", "c"})
	require.Equal(t, SliceRemove([]string{"a", "b", "c"}, "b"), []string{"a", "c"})
	require.Equal(t, SliceRemove([]string{"a", "b", "c"}, "c"), []string{"a", "b"})
	require.Equal(t, SliceRemove([]string{"a", "b", "c"}, "a"), []string{"b", "c"})
}

func TestSliceDelete(t *testing.T) {
	require.Equal(t, SliceDelete([]string{"a"}, 0), []string{})
	require.Equal(t, SliceDelete([]string{"a", "b"}, 0), []string{"b"})
	require.Equal(t, SliceDelete([]string{"a", "b"}, 1), []string{"a"})
	require.Equal(t, SliceDelete([]string{"a", "b", "c"}, 0), []string{"b", "c"})
	require.Equal(t, SliceDelete([]string{"a", "b", "c"}, 1), []string{"a", "c"})
	require.Equal(t, SliceDelete([]string{"a", "b", "c"}, 2), []string{"a", "b"})
	require.Equal(t, SliceDelete([]string{"a", "b", "c"}, 0), []string{"b", "c"})
}
