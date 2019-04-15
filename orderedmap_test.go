package common

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestDefault(t *testing.T) {
	o := NewOrderedMap()

	for i := 0; i < 100; i++ {
		o.Set(i, strconv.Itoa(i))
	}

	for i := 0; i < 100; i++ {
		v, ok := o.Get(i)

		assert.True(t, ok)
		assert.Equal(t, strconv.Itoa(i), v.(string))
	}

	for i, k := range o.Keys() {
		assert.Equal(t, i, k.(int))
	}
}

func TestFilled(t *testing.T) {
	m := make(map[int]string)

	for i := 0; i < 100; i++ {
		m[i] = strconv.Itoa(i)
	}

	o := NewOrderedMap(m)

	for i := 0; i < 10; i++ {
		var k int
		for {
			k = Rnd(100)

			_, ok := m[k]

			if ok {
				break
			}
		}

		delete(m, k)
		o.Delete(k)
	}

	for _, k := range o.Keys() {
		v, ok := o.Get(k)

		assert.True(t, ok)
		assert.Equal(t, m[k.(int)], v)
	}
}
