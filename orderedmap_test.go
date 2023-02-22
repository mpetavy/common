package common

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestDefault(t *testing.T) {
	orderedMap := NewOrderedMap[int, string]()

	for i := 0; i < 5; i++ {
		orderedMap.Add(i, strconv.Itoa(i))
	}

	for i := 0; i < 5; i++ {
		v, ok := orderedMap.Get(i)

		assert.True(t, ok)
		assert.Equal(t, strconv.Itoa(i), v)
	}

	keys := orderedMap.Keys()

	for i, k := range keys {
		assert.Equal(t, i, k)
	}
}

func TestFilled(t *testing.T) {
	m := make(map[int]string)
	orderedMap := NewOrderedMap[int, string]()

	for i := 0; i < 10; i++ {
		v := strconv.Itoa(i)
		m[i] = v
		orderedMap.Add(i, v)
	}

	for i := 0; i < 10; i++ {
		var k int
		for {
			k = Rnd(10)

			_, ok := m[k]
			if ok {
				break
			}
		}

		delete(m, k)
		orderedMap.Remove(k)
	}

	for _, k := range orderedMap.Keys() {
		v, ok := orderedMap.Get(k)

		assert.True(t, ok)
		assert.Equal(t, m[k], v)
	}
}

func TestIndex(t *testing.T) {
	orderedMap := NewOrderedMap[int, string]()

	orderedMap.Add(0, "00")
	orderedMap.Add(1, "11")
	orderedMap.Add(2, "22")

	index, ok := orderedMap.Get(1)

	assert.True(t, ok)
	assert.Equal(t, "11", index)

	v := orderedMap.Remove(1)

	assert.Equal(t, "11", v)
	assert.Equal(t, 2, orderedMap.Len())

	k, v := orderedMap.GetByIndex(0)
	assert.Equal(t, 0, k)
	assert.Equal(t, "00", v)

	k, v = orderedMap.GetByIndex(1)
	assert.Equal(t, 2, k)
	assert.Equal(t, "22", v)
}
