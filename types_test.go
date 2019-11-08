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

func fillNameOfStruct(data interface{}) {
	adr := data.(*InnerStruct)
	adr.InnerField = "bbb"
}

func TestIterateStruct(t *testing.T) {
	s := OuterStruct{}

	err := IterateStruct(&s, func(typ reflect.StructField, val reflect.Value) error {
		switch val.Type().Kind() {
		case reflect.Struct:
			fillNameOfStruct(val.Addr().Interface())
		default:
			if val.Type().Kind() == reflect.String {
				if val.String() == "" {
					val.SetString("aaa")
				}
			}
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
