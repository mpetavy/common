package common

import (
	"io"
	"time"
)

type throttledWriter struct {
	writer          io.Writer
	bytesPerSeconds int
}

func (w *throttledWriter) Write(p []byte) (int, error) {
	if w.bytesPerSeconds == 0 {
		return w.writer.Write(p)
	}

	index := 0

	for {
		amount := Min(w.bytesPerSeconds, len(p)-index)

		timestamp := time.Now()
		n, err := WriteFully(w.writer, p[index:index+amount])

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

func NewThrottledWriter(writer io.Writer, bytesPerSeconds int) io.Writer {
	return &throttledWriter{
		writer:          writer,
		bytesPerSeconds: bytesPerSeconds,
	}
}
