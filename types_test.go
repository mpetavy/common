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
	assert.True(t, ew(t, "TEST.GO", "test.go"))
	assert.False(t, ew(t, "TEST.GO", "test.goo"))
	assert.False(t, ew(t, "TEST.GO", "test.go?"))
	assert.True(t, ew(t, "TEST.GO", "test.go*"))
	assert.True(t, ew(t, "TEST.GO", "*.go"))
	assert.True(t, ew(t, "TEST.GO", "test.*"))
	assert.True(t, ew(t, "TEST.GO", "??st.go"))
	assert.True(t, ew(t, "TEST.GO", "test.??"))
	assert.True(t, ew(t, "?", "?"))
	assert.True(t, ew(t, "?", "?"))
	assert.True(t, ew(t, "CFG.FILE", "cfg.file*"))
	assert.True(t, ew(t, "CFG.FILE.TEMPLATE", "cfg.file*"))
	assert.True(t, ew(t, "?MD", "?md"))
	assert.False(t, ew(t, "?MD", "?.md"))
	assert.False(t, ew(t, "?MD", "\\?md"))

	masks := []string{
		FlagNameCfgFile + "*",
		FlagNameCfgReset,
		FlagNameCfgCreate,
		FlagNameUsage,
		FlagNameUsageMd,
		"test*",
	}

	for _, mask := range masks {
		assert.True(t, ew(t, mask, mask), mask)
	}

	assert.True(t, ew(t, "cfg.file", "cfg.file*"))
	assert.True(t, ew(t, "cfg.file.template", "cfg.file*"))

	assert.False(t, ew(t, "cfg.file", "xxx"))
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
