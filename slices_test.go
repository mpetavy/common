package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSliceClone(t *testing.T) {
	assert.Equal(t, SliceClone([]string{}), []string{})
	assert.Equal(t, SliceClone([]string{"a"}), []string{"a"})
	assert.Equal(t, SliceClone([]string{"a", "b", "c"}), []string{"a", "b", "c"})

	a := []string{"a"}
	b := SliceClone(a)

	a[0] = "x"

	assert.Equal(t, a, []string{"x"})
	assert.Equal(t, b, []string{"a"})
}

func TestSliceIndex(t *testing.T) {
	assert.Equal(t, SliceIndex([]string{}, "a"), -1)
	assert.Equal(t, SliceIndex([]string{"a"}, "a"), 0)
	assert.Equal(t, SliceIndex([]string{"a", "b"}, "a"), 0)
	assert.Equal(t, SliceIndex([]string{"a", "b"}, "b"), 1)
	assert.Equal(t, SliceIndex([]string{"a", "b"}, "x"), -1)
}

func TestSliceIndexFunc(t *testing.T) {
	assert.Equal(t, SliceIndexFunc([]string{}, func(e string) bool {
		return false
	}), -1)
	assert.Equal(t, SliceIndexFunc([]string{"x"}, func(e string) bool {
		return e == "x"
	}), 0)
	assert.Equal(t, SliceIndexFunc([]string{"a", "x", "b"}, func(e string) bool {
		return e == "x"
	}), 1)
	assert.Equal(t, SliceIndexFunc([]string{"a", "b", "x"}, func(e string) bool {
		return e == "x"
	}), 2)
	assert.Equal(t, SliceIndexFunc([]string{"a", "b", "c"}, func(e string) bool {
		return e == "x"
	}), -1)
}

func TestSliceAppend(t *testing.T) {
	assert.Equal(t, SliceAppend([]string{}, "a"), []string{"a"})
	assert.Equal(t, SliceAppend([]string{"a"}, "b"), []string{"a", "b"})
	assert.Equal(t, SliceAppend([]string{"a"}, "b", "c"), []string{"a", "b", "c"})
}

func TestSliceRemove(t *testing.T) {
	assert.Equal(t, SliceRemove([]string{}, "a"), []string{})
	assert.Equal(t, SliceRemove([]string{"a"}, "a"), []string{})
	assert.Equal(t, SliceRemove([]string{"a", "b"}, "a"), []string{"b"})
	assert.Equal(t, SliceRemove([]string{"a", "b"}, "b"), []string{"a"})
	assert.Equal(t, SliceRemove([]string{"a", "b", "c"}, "a"), []string{"b", "c"})
	assert.Equal(t, SliceRemove([]string{"a", "b", "c"}, "b"), []string{"a", "c"})
	assert.Equal(t, SliceRemove([]string{"a", "b", "c"}, "c"), []string{"a", "b"})
	assert.Equal(t, SliceRemove([]string{"a", "b", "c"}, "a"), []string{"b", "c"})
}

func TestSliceInsert(t *testing.T) {
	assert.Equal(t, SliceInsert([]string{}, 0), []string{})
	assert.Equal(t, SliceInsert([]string{}, 0, "x"), []string{"x"})
	assert.Equal(t, SliceInsert([]string{"a"}, 0, "x"), []string{"x", "a"})
	assert.Equal(t, SliceInsert([]string{"a", "b"}, 0, "x"), []string{"x", "a", "b"})
	assert.Equal(t, SliceInsert([]string{"a", "b"}, 1, "x"), []string{"a", "x", "b"})
	assert.Equal(t, SliceInsert([]string{"a", "b"}, 2, "x"), []string{"a", "b", "x"})
	assert.Equal(t, SliceInsert([]string{"a", "b"}, 1, "x", "y"), []string{"a", "x", "y", "b"})
}

func TestSliceDelete(t *testing.T) {
	assert.Equal(t, SliceDelete([]string{"a"}, 0), []string{})
	assert.Equal(t, SliceDelete([]string{"a", "b"}, 0), []string{"b"})
	assert.Equal(t, SliceDelete([]string{"a", "b"}, 1), []string{"a"})
	assert.Equal(t, SliceDelete([]string{"a", "b", "c"}, 0), []string{"b", "c"})
	assert.Equal(t, SliceDelete([]string{"a", "b", "c"}, 1), []string{"a", "c"})
	assert.Equal(t, SliceDelete([]string{"a", "b", "c"}, 2), []string{"a", "b"})
	assert.Equal(t, SliceDelete([]string{"a", "b", "c"}, 0), []string{"b", "c"})
}

func TestSliceDeleteRange(t *testing.T) {
	assert.Equal(t, SliceDeleteRange([]string{"a", "b"}, 0, 2), []string{})
	assert.Equal(t, SliceDeleteRange([]string{"a", "b"}, 0, 1), []string{"b"})
	assert.Equal(t, SliceDeleteRange([]string{"a", "b"}, 1, 1), []string{"a", "b"})
	assert.Equal(t, SliceDeleteRange([]string{"a", "b"}, 1, 2), []string{"a"})
	assert.Equal(t, SliceDeleteRange([]string{"a", "b", "c"}, 0, 2), []string{"c"})
	assert.Equal(t, SliceDeleteRange([]string{"a", "b", "c"}, 1, 2), []string{"a", "c"})
	assert.Equal(t, SliceDeleteRange([]string{"a", "b", "c"}, 2, 3), []string{"a", "b"})
	assert.Equal(t, SliceDeleteRange([]string{"a", "b", "c"}, 0, 3), []string{})
}
