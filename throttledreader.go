package common

import (
	"io"
	"time"
)

type throttledReader struct {
	reader          io.Reader
	bytesPerSeconds int
}

func (w *throttledReader) Read(p []byte) (int, error) {
	if w.bytesPerSeconds == 0 {
		return w.reader.Read(p)
	}

	index := 0

	for {
		amount := Min(w.bytesPerSeconds, len(p)-index)

		timestamp := time.Now()
		n, err := ReadFully(w.reader, p[index:index+amount])

		index += n

		if err != nil {
			return index, err
		}

		if index == len(p) {
			return index, nil
		}

		sleepTime := timestamp.Add(time.Second).Sub(time.Now())
		if sleepTime > 0 {
			time.Sleep(sleepTime)
		}
	}
}

func NewThrottledReader(reader io.Reader, bytesPerSeconds int) io.Reader {
	return &throttledReader{
		reader:          reader,
		bytesPerSeconds: bytesPerSeconds,
	}
}
