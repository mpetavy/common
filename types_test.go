package common

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestContainsWildcard(t *testing.T) {
	assert.True(t, ContainsWildcard("a?b"))
	assert.True(t, ContainsWildcard("a*b"))
	assert.False(t, ContainsWildcard("ab"))
}

func ew(t *testing.T, s string, m string) bool {
	b, err := EqualWildcards(s, m)
	if Error(err) {
		t.Fail()
	}

	return b
}

func TestEqualWildcards(t *testing.T) {
	assert.True(t, ew(t, "test.go", "test.go"))
	assert.False(t, ew(t, "test.go", "test.goo"))
	assert.False(t, ew(t, "test.go", "test.go?"))
	assert.True(t, ew(t, "test.go", "test.go*"))
	assert.True(t, ew(t, "test.go", "*.go"))
	assert.True(t, ew(t, "test.go", "test.*"))
	assert.True(t, ew(t, "test.go", "??st.go"))
	assert.True(t, ew(t, "test.go", "test.??"))
	assert.True(t, ew(t, "?", "?"))
	assert.True(t, ew(t, ("?"), "?"))
	assert.True(t, ew(t, ("cfg.file"), "cfg.file*"))
	assert.True(t, ew(t, ("cfg.file.template"), "cfg.file*"))
	assert.True(t, ew(t, ("?md"), "?md"))
	assert.False(t, ew(t, ("?md"), "?.md"))
	assert.True(t, ew(t, ("?md"), "\\?md"))
}

func TestPersistWildcards(t *testing.T) {
	assert.Equal(t, "\\?\\?", PersistWildcards("??"))
	assert.Equal(t, "\\*\\*", PersistWildcards("**"))
	assert.Equal(t, "\\.\\.", PersistWildcards(".."))
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
