package common

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestDefault(t *testing.T) {
	orderedMap := NewOrderedMap[int, string]()

	for i := 0; i < 100; i++ {
		orderedMap.Add(i, strconv.Itoa(i))
	}

	for i := 0; i < 100; i++ {
		v, ok := orderedMap.GetOk(i)

		assert.True(t, ok)
		assert.Equal(t, strconv.Itoa(i), v)
	}

	for i, k := range orderedMap.Keys() {
		assert.Equal(t, i, k)
	}
}

func TestFilled(t *testing.T) {
	mab := make(map[int]string)
	orderedMap := NewOrderedMap[int, string]()

	for i := 0; i < 100; i++ {
		v := strconv.Itoa(i)
		mab[i] = v
		orderedMap.Add(i, v)
	}

	for i := 0; i < 10; i++ {
		var k int
		for {
			k = Rnd(100)

			_, ok := mab[k]
			if ok {
				break
			}
		}

		delete(mab, k)
		orderedMap.Remove(k)
	}

	for _, k := range orderedMap.Keys() {
		v, ok := orderedMap.GetOk(k)

		assert.True(t, ok)
		assert.Equal(t, mab[k], v)
	}
}
