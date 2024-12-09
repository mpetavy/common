package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSliceClone(t *testing.T) {
	require.Equal(t, SliceClone([]string{}), []string{})
	require.Equal(t, SliceClone([]string{"a"}), []string{"a"})
	require.Equal(t, SliceClone([]string{"a", "b", "c"}), []string{"a", "b", "c"})

	a := []string{"a"}
	b := SliceClone(a)

	a[0] = "x"

	require.Equal(t, a, []string{"x"})
	require.Equal(t, b, []string{"a"})
}

func TestSliceIndex(t *testing.T) {
	require.Equal(t, SliceIndex([]string{}, "a"), -1)
	require.Equal(t, SliceIndex([]string{"a"}, "a"), 0)
	require.Equal(t, SliceIndex([]string{"a", "b"}, "a"), 0)
	require.Equal(t, SliceIndex([]string{"a", "b"}, "b"), 1)
	require.Equal(t, SliceIndex([]string{"a", "b"}, "x"), -1)
}

func TestSliceIndexFunc(t *testing.T) {
	require.Equal(t, SliceIndexFunc([]string{}, func(e string) bool {
		return false
	}), -1)
	require.Equal(t, SliceIndexFunc([]string{"x"}, func(e string) bool {
		return e == "x"
	}), 0)
	require.Equal(t, SliceIndexFunc([]string{"a", "x", "b"}, func(e string) bool {
		return e == "x"
	}), 1)
	require.Equal(t, SliceIndexFunc([]string{"a", "b", "x"}, func(e string) bool {
		return e == "x"
	}), 2)
	require.Equal(t, SliceIndexFunc([]string{"a", "b", "c"}, func(e string) bool {
		return e == "x"
	}), -1)
}

func TestSliceAppend(t *testing.T) {
	require.Equal(t, SliceAppend([]string{}, "a"), []string{"a"})
	require.Equal(t, SliceAppend([]string{"a"}, "b"), []string{"a", "b"})
	require.Equal(t, SliceAppend([]string{"a"}, "b", "c"), []string{"a", "b", "c"})
}

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

func TestSliceInsert(t *testing.T) {
	require.Equal(t, SliceInsert([]string{}, 0), []string{})
	require.Equal(t, SliceInsert([]string{}, 0, "x"), []string{"x"})
	require.Equal(t, SliceInsert([]string{"a"}, 0, "x"), []string{"x", "a"})
	require.Equal(t, SliceInsert([]string{"a", "b"}, 0, "x"), []string{"x", "a", "b"})
	require.Equal(t, SliceInsert([]string{"a", "b"}, 1, "x"), []string{"a", "x", "b"})
	require.Equal(t, SliceInsert([]string{"a", "b"}, 2, "x"), []string{"a", "b", "x"})
	require.Equal(t, SliceInsert([]string{"a", "b"}, 1, "x", "y"), []string{"a", "x", "y", "b"})
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

func TestSliceDeleteRange(t *testing.T) {
	require.Equal(t, SliceDeleteRange([]string{"a", "b"}, 0, 2), []string{})
	require.Equal(t, SliceDeleteRange([]string{"a", "b"}, 0, 1), []string{"b"})
	require.Equal(t, SliceDeleteRange([]string{"a", "b"}, 1, 1), []string{"a", "b"})
	require.Equal(t, SliceDeleteRange([]string{"a", "b"}, 1, 2), []string{"a"})
	require.Equal(t, SliceDeleteRange([]string{"a", "b", "c"}, 0, 2), []string{"c"})
	require.Equal(t, SliceDeleteRange([]string{"a", "b", "c"}, 1, 2), []string{"a", "c"})
	require.Equal(t, SliceDeleteRange([]string{"a", "b", "c"}, 2, 3), []string{"a", "b"})
	require.Equal(t, SliceDeleteRange([]string{"a", "b", "c"}, 0, 3), []string{})
}
