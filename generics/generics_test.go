package generics

import (
	"container/list"
	"github.com/stretchr/testify/assert"
	"testing"
)

type TestData struct {
	comparator Comparator
	values     [3]interface{}
}

func TestList(t *testing.T) {
	var tds = [...]TestData{{StringComparator(), [3]interface{}{"a", "b", "c"}}, {IntegerComparator(), [3]interface{}{0, 1, 2}}}

	for _, td := range tds {
		var l list.List

		index, err := FindInList(l, td.values[0], td.comparator)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, index, -1)

		for _, item := range td.values {
			l.PushBack(item)
		}

		for i := range td.values {
			index, err := FindInList(l, td.values[i], td.comparator)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, index, index)
		}
	}
}

func TestSlice(t *testing.T) {
	var tds = [...]TestData{{StringComparator(), [3]interface{}{"a", "b", "c"}}, {IntegerComparator(), [3]interface{}{0, 1, 2}}}

	for _, td := range tds {
		slice := td.values[:]
		slice = nil

		index, err := FindInSlice(slice, td.values[0], td.comparator)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, index, -1)

		slice = td.values[:]

		for i := range td.values {
			index, err := FindInSlice(slice, td.values[i], td.comparator)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, index, index)
		}
	}
}
