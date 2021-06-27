package common

import (
	"io"
)

type sizedReader struct {
	reader io.Reader
	read   int64
	size   int64
}

func (this *sizedReader) Read(p []byte) (n int, err error) {
	if this.read < this.size {
		r, err := this.reader.Read(p)
		if err != nil {
			return r, err
		}

		toMuch := this.size - (this.read + int64(r))
		if toMuch <= 0 {
			p = p[0 : int64(r)+toMuch]
		}

		this.read += int64(r)

		return len(p), nil
	}

	return 0, io.EOF
}

func NewSizedReader(reader io.Reader, size int64) io.Reader {
	r := &sizedReader{
		reader: reader,
		size:   size,
	}

	return r
}
