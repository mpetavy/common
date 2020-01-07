package common

import (
	"io"
	"time"
)

type throttledReader struct {
	reader       io.Reader
	bytesPerSecs int
}

func (this *throttledReader) Read(p []byte) (int, error) {
	start := time.Now()

	n, err := this.reader.Read(p)

	if this.bytesPerSecs > 0 {
		mustMilli := int(float64(n) / float64(this.bytesPerSecs) * 1000)

		target := start.Add(time.Millisecond * time.Duration(mustMilli))

		//d := target.Sub(time.Now())
		d := target.Sub(start)
		if d > 0 {
			time.Sleep(d)
		}
	}

	return n, err
}

func NewThrottledReader(reader io.Reader, bytesPerSecs int) io.Reader {
	r := &throttledReader{
		reader:       reader,
		bytesPerSecs: bytesPerSecs,
	}

	return r
}
