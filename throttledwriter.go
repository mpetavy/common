package common

import (
	"io"
	"time"
)

type throttledWriter struct {
	writer       io.Writer
	bytesPerSecs int
	lastSec      time.Time
	written      int
}

func (this *throttledWriter) Write(p []byte) (n int, err error) {
	if this.bytesPerSecs == 0 {
		return this.writer.Write(p)
	}

	curSec := time.Now()
	lenP := len(p)
	index := 0

	var sleep bool

	for {
		sleep = false

		if this.lastSec.IsZero() || this.lastSec.Before(curSec) {
			this.lastSec = curSec
			this.written = 0
		}

		remainAllowed := this.bytesPerSecs - this.written
		writeLen := lenP - index

		if writeLen > remainAllowed {
			writeLen = remainAllowed
			sleep = true
		}

		n, err := this.writer.Write(p[index : index+writeLen])
		index += n

		if err != nil {
			return index, err
		}

		if index == lenP {
			break
		}

		if sleep {
			curSec = this.lastSec.Add(time.Second)
			d := curSec.Sub(time.Now())

			if d > 0 {
				time.Sleep(d)
			}
		}
	}

	return index, nil
}

func NewThrottledWriter(writer io.Writer, bytesPerSecs int) io.Writer {
	lw := &throttledWriter{
		writer:       writer,
		bytesPerSecs: bytesPerSecs,
	}

	return lw
}
