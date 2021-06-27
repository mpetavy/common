package common

import (
	"bytes"
	"io"
	"testing"
)

func TestSizedReader(t *testing.T) {
	type args struct {
		reader io.Reader
		size   int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "small stream",
			args: args{
				reader: bytes.NewReader([]byte("")),
				size:   10,
			},
			want: 0,
		},
		{
			name: "normal stream",
			args: args{
				reader: bytes.NewReader([]byte("Hello")),
				size:   5,
			},
			want: 5,
		},
		{
			name: "big stream",
			args: args{
				reader: bytes.NewReader([]byte("Hello world")),
				size:   5,
			},
			want: 5,
		},
		{
			name: "very big stream",
			args: args{
				reader: bytes.NewReader(make([]byte, 10000)),
				size:   9000,
			},
			want: 9000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewSizedReader(tt.args.reader, tt.args.size)

			r, err := io.Copy(io.Discard, reader)

			if err != nil && err != io.EOF {
				t.Errorf(err.Error())
			}

			if r != tt.want {
				t.Errorf("NewSizedReader() = %v, want %v", r, tt.want)
			}
		})
	}
}
