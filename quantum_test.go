package common

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func check(t *testing.T, q *Quantum, ints []int) {
	sl := q.ToSlice()

	assert.True(t, q.Len() == len(ints), "Len of quantum")
	assert.Equal(t, sl, ints, "Content of quantum")

	for i := 0; i < q.Len(); i++ {
		v, err := q.Get(i)

		if err != nil {
			t.Error(err)
		}
		if v != sl[i] {
			fmt.Printf("%s\n", q)
			assert.Equal(t, v, sl[i], "Get() shows different value on index %d", i)
		}
	}
}

func TestQuantum(t *testing.T) {
	q := NewQuantum()

	check(t, q, []int{})

	q.Add(0)
	check(t, q, []int{0})

	q.Add(2)
	q.Add(4)
	check(t, q, []int{0, 2, 4})

	q.Add(1)
	check(t, q, []int{0, 1, 2, 4})

	q.Add(99)
	check(t, q, []int{0, 1, 2, 4, 99})

	q.Add(2)
	check(t, q, []int{0, 1, 2, 4, 99})

	q.Remove(2)
	check(t, q, []int{0, 1, 4, 99})

	q.Remove(0)
	check(t, q, []int{1, 4, 99})

	q.Remove(99)
	check(t, q, []int{1, 4})

	q.Remove(4)
	check(t, q, []int{1})

	q.Remove(1)
	check(t, q, []int{})

	q, err := ParseQuantum("3;5;5-7;27;123")
	if err != nil {
		t.Error(err)
	}
	check(t, q, []int{3, 5, 6, 7, 27, 123})
}

func TestQuantumLoop(t *testing.T) {
	count := 10000
	sl := []int{}
	q := NewQuantum()

	q.AddRange(0, count-1)

	for i := 0; i < count; i++ {
		index := 0
		if q.Len() > 1 {
			index = Rnd(q.Len() - 1)
		}

		value, err := q.Get(index)
		if err != nil {
			t.Error(err)
		}

		q.Remove(value)

		sl = append(sl, value)
	}

	assert.Equal(t, 0, q.Len())

	for i := 0; i < count; i++ {
		if sort.SearchInts(sl, i) == -1 {
			t.Fatalf("not found: %d", i)
		}
	}
}
