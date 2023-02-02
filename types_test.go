package common

import (
	"bufio"
	"bytes"
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

func join(bas ...[]byte) []byte {
	buf := bytes.Buffer{}

	for _, ba := range bas {
		buf.Write(ba)
	}

	return buf.Bytes()
}

func TestNewSeparatorSplitFunc(t *testing.T) {
	InitTesting(t)

	hello := []byte("hello")
	//word := []byte("world")
	prefix := []byte(">>>")
	suffix := []byte("<<<")
	//rndBytes, err := RndBytes(10)
	//if common.Error(err) {
	//	return
	//}

	type args struct {
		prefix []byte
		suffix []byte
		remove bool
	}
	tests := []struct {
		name string
		args args
		data []byte
		want []byte
	}{
		{
			name: "0",
			args: args{},
			data: nil,
			want: nil,
		},
		{
			name: "1",
			args: args{
				prefix: prefix,
			},
			data: nil,
			want: nil,
		},
		{
			name: "2",
			args: args{
				suffix: suffix,
			},
			data: nil,
			want: nil,
		},
		{
			name: "3",
			args: args{
				suffix: suffix,
				remove: false,
			},
			data: join(prefix, suffix),
			want: []byte(""),
		},
		{
			name: "4",
			args: args{
				suffix: suffix,
				remove: false,
			},
			data: join(hello, suffix),
			want: join(hello, suffix),
		},
		//{
		//	name:      "2",
		//	args:      join(Hl7Start),
		//	wantToken: nil,
		//	wantErr:   false,
		//},
		//{
		//	name:      "3",
		//	args:      join(Hl7End),
		//	wantToken: nil,
		//	wantErr:   false,
		//},
		//{
		//	name:      "4",
		//	args:      join(Hl7End, Hl7Start),
		//	wantToken: nil,
		//	wantErr:   false,
		//},
		//{
		//	name:      "4",
		//	args:      join(Hl7Start, Hl7End),
		//	wantToken: join(Hl7Start, Hl7End),
		//	wantErr:   false,
		//},
		//{
		//	name:      "4",
		//	args:      join(rndBytes(10), Hl7Start, Hl7End, rndBytes(10)),
		//	wantToken: join(Hl7Start, Hl7End),
		//	wantErr:   false,
		//},
		//{
		//	name:      "5",
		//	args:      join(rndBytes(10), Hl7Start, []byte("hello"), Hl7End, rndBytes(10), Hl7Start, []byte("world"), Hl7End, rndBytes(10)),
		//	wantToken: join(Hl7Start, []byte("hello"), Hl7End, Hl7Start, []byte("world"), Hl7End),
		//	wantErr:   false,
		//},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := bufio.NewScanner(bytes.NewReader(tt.data))
			scanner.Split(NewSeparatorSplitFunc(tt.args.prefix, tt.args.suffix, tt.args.remove))

			buf := bytes.Buffer{}

			for scanner.Scan() {
				buf.Write(scanner.Bytes())
			}

			assert.Equalf(t, tt.want, buf.Bytes(), "NewSeparatorSplitFunc(%v, %v, %v)", tt.args.prefix, tt.args.suffix, tt.args.remove)
		})
	}
}
