package common

import (
	"io"
	"time"
)

type throttledWriter struct {
	writer         io.Writer
	bytesPerSecs   int
	lastSec        time.Time
	lastSecWritten int
}

func (this *throttledWriter) Write(p []byte) (n int, err error) {
	if this.bytesPerSecs == 0 {
		return this.writer.Write(p)
	}

	lenP := len(p)
	index := 0

	curSec := time.Now().Truncate(time.Second)
	if this.lastSec.IsZero() || this.lastSec.Before(curSec) {
		this.lastSec = curSec
		this.lastSecWritten = 0
	}

	for {
		sleep := false

		remainAllowedLen := this.bytesPerSecs - this.lastSecWritten
		remainBufferLen := lenP - index

		if remainBufferLen > remainAllowedLen {
			remainBufferLen = remainAllowedLen
			sleep = true
		}

		n, err := this.writer.Write(p[index : index+remainBufferLen])

		index += n
		this.lastSecWritten += n

		if Error(err) {
			return index, err
		}

		if index == lenP {
			return index, nil
		}

		if sleep {
			this.lastSec = this.lastSec.Add(time.Second)
			this.lastSecWritten = 0

			d := this.lastSec.Sub(time.Now())

			if d > 0 {
				Sleep(d)
			}
		}
	}
}

func NewThrottledWriter(writer io.Writer, bytesPerSecs int) io.Writer {
	w := &throttledWriter{
		writer:       writer,
		bytesPerSecs: bytesPerSecs,
	}

	return w
}
