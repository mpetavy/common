package common

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestContainsWildcard(t *testing.T) {
	b := ContainsWildcard("a?b")
	if !b {
		t.Fail()
	}

	b = ContainsWildcard("a*b")
	if !b {
		t.Fail()
	}

	b = ContainsWildcard("ab")
	if b {
		t.Fail()
	}
}

func TestEqualWildcards(t *testing.T) {
	b, err := EqualWildcards("test.go", "test.go")
	if !b || err != nil {
		t.Fail()
	}

	b, err = EqualWildcards("test.go", "test.goo")
	if b || err != nil {
		t.Fail()
	}

	b, err = EqualWildcards("test.go", "*.go")
	if !b || err != nil {
		t.Fail()
	}

	b, err = EqualWildcards("test.go", "test.*")
	if !b || err != nil {
		t.Fail()
	}

	b, err = EqualWildcards("test.go", "??st.go")
	if !b || err != nil {
		t.Fail()
	}

	b, err = EqualWildcards("test.go", "test.??")
	if !b || err != nil {
		t.Fail()
	}
}

type InnerStruct struct {
	InnerField string
}

type OuterStruct struct {
	OuterField string
	Tel        InnerStruct
}

func TestIterateStruct(t *testing.T) {
	s := OuterStruct{}

	err := IterateStruct(&s, func(fieldPath string, fieldType reflect.StructField, fieldValue reflect.Value) error {
		outer, ok := fieldValue.Addr().Interface().(*OuterStruct)
		if ok {
			outer.OuterField = "aaa"
		}
		inner, ok := fieldValue.Addr().Interface().(*InnerStruct)
		if ok {
			inner.InnerField = "bbb"
		}

		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, &OuterStruct{
		OuterField: "aaa",
		Tel: InnerStruct{
			InnerField: "bbb",
		}}, &s)
}
