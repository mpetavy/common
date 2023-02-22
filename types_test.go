package common

import (
	"bufio"
	"bytes"
	"fmt"
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
	world := []byte("world")
	prefix := []byte(">>>")
	suffix := []byte("<<<")
	noiseStr, err := RndString(10)
	if Error(err) {
		return
	}
	noise := []byte(noiseStr)

	type args struct {
		prefix []byte
		suffix []byte
		remove bool
	}
	tests := []struct {
		name    string
		args    args
		data    []byte
		want    []byte
		wantErr bool
	}{
		{
			name:    "0",
			args:    args{},
			data:    nil,
			want:    nil,
			wantErr: true,
		},
		{
			name: "1",
			args: args{
				prefix: prefix,
				suffix: suffix,
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
			},
			data: (suffix),
			want: suffix,
		},
		{
			name: "4",
			args: args{
				suffix: suffix,
				remove: true,
			},
			data: join(suffix),
			want: nil,
		},
		{
			name: "5",
			args: args{
				suffix: suffix,
			},
			data: join(hello, suffix),
			want: join(hello, suffix),
		},
		{
			name: "6",
			args: args{
				suffix: suffix,
				remove: true,
			},
			data: join(hello, suffix),
			want: hello,
		},
		{
			name: "7",
			args: args{
				prefix: prefix,
				suffix: suffix,
			},
			data: nil,
			want: nil,
		},
		{
			name: "8",
			args: args{
				prefix: prefix,
				suffix: suffix,
			},
			data: prefix,
			want: nil,
		},
		{
			name: "9",
			args: args{
				prefix: prefix,
				suffix: suffix,
			},
			data: suffix,
			want: nil,
		},
		{
			name: "10",
			args: args{
				prefix: prefix,
				suffix: suffix,
			},
			data: join(prefix, suffix),
			want: join(prefix, suffix),
		},
		{
			name: "11",
			args: args{
				prefix: prefix,
				suffix: suffix,
				remove: true,
			},
			data: join(prefix, suffix),
			want: nil,
		},
		{
			name: "12",
			args: args{
				prefix: prefix,
				suffix: suffix,
			},
			data: join(suffix, prefix),
			want: nil,
		},
		{
			name: "13",
			args: args{
				prefix: prefix,
				suffix: suffix,
				remove: true,
			},
			data: join(suffix, prefix),
			want: nil,
		},
		{
			name: "14",
			args: args{
				prefix: prefix,
				suffix: suffix,
			},
			data: join(prefix, hello, suffix),
			want: join(prefix, hello, suffix),
		},
		{
			name: "15",
			args: args{
				prefix: prefix,
				suffix: suffix,
				remove: true,
			},
			data: join(prefix, hello, suffix),
			want: hello,
		},
		{
			name: "16",
			args: args{
				prefix: prefix,
				suffix: suffix,
			},
			data: join(noise, prefix, hello, suffix),
			want: join(prefix, hello, suffix),
		},
		{
			name: "17",
			args: args{
				prefix: prefix,
				suffix: suffix,
				remove: true,
			},
			data: join(noise, prefix, hello, suffix),
			want: hello,
		},
		{
			name: "18",
			args: args{
				prefix: prefix,
				suffix: suffix,
			},
			data: join(noise, prefix, hello, suffix, noise, prefix, world, suffix),
			want: join(prefix, hello, suffix, prefix, world, suffix),
		},
		{
			name: "19",
			args: args{
				prefix: prefix,
				suffix: suffix,
				remove: true,
			},
			data: join(noise, prefix, hello, suffix, noise, prefix, world, suffix),
			want: join(hello, world),
		},
		{
			name: "20",
			args: args{
				prefix: prefix,
				suffix: suffix,
				remove: true,
			},
			data: join(noise, prefix, hello, suffix, noise, prefix, world, suffix, noise),
			want: join(hello, world),
		},
		{
			name:    "21",
			args:    args{},
			wantErr: true,
		},
		{
			name: "22",
			args: args{
				prefix: nil,
				suffix: []byte("\n"),
				remove: true,
			},
			data: []byte(fmt.Sprintf("%s\n%s\n", hello, world)),
			want: []byte(fmt.Sprintf("%s%s", hello, world)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sf, err := NewSeparatorSplitFunc(tt.args.prefix, tt.args.suffix, tt.args.remove)

			assert.Equal(t, tt.wantErr, err != nil)

			if tt.wantErr {
				return
			}

			scanner := bufio.NewScanner(bytes.NewReader(tt.data))
			scanner.Split(sf)

			buf := bytes.Buffer{}

			for scanner.Scan() {
				buf.Write(scanner.Bytes())
			}

			assert.Equalf(t, tt.want, buf.Bytes(), "NewSplitFuncSeparator(%v, %v, %v)", tt.args.prefix, tt.args.suffix, tt.args.remove)
		})
	}
}

func TestReverseSlice(t *testing.T) {
	type args[T any] struct {
		original []T
	}
	type testCase[T any] struct {
		name string
		args args[T]
		want []T
	}
	tests := []testCase[int]{
		{
			name: "0",
			args: args[int]{
				original: []int{},
			},
			want: []int{},
		},
		{
			name: "1",
			args: args[int]{
				original: []int{1},
			},
			want: []int{1},
		},
		{
			name: "2",
			args: args[int]{
				original: []int{1, 2, 3},
			},
			want: []int{3, 2, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, ReverseSlice(tt.args.original), "ReverseSlice(%v)", tt.args.original)
		})
	}
}

func TestRemoveSlice(t *testing.T) {
	type args[T comparable] struct {
		slice []T
		item  T
		count int
	}
	type testCase[T comparable] struct {
		name string
		args args[T]
		want []T
	}
	tests := []testCase[int]{
		{
			name: "0",
			args: args[int]{
				slice: []int{},
				item:  2,
				count: -1,
			},
			want: []int{},
		},
		{
			name: "1",
			args: args[int]{
				slice: []int{1},
				item:  1,
				count: 0,
			},
			want: []int{1},
		},
		{
			name: "2",
			args: args[int]{
				slice: []int{1},
				item:  1,
				count: -1,
			},
			want: []int{},
		},
		{
			name: "3",
			args: args[int]{
				slice: []int{1, 2, 1},
				item:  1,
				count: 2,
			},
			want: []int{2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, RemoveSlice(tt.args.slice, tt.args.item, tt.args.count), "RemoveSlice(%v, %v, %v)", tt.args.slice, tt.args.item, tt.args.count)
		})
	}
}
