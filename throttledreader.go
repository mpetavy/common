package common

import (
	"io"
	"time"
)

type throttledReader struct {
	reader       io.Reader
	bytesPerSecs int
	lastSec      time.Time
	lastSecRead  int
}

func (this *throttledReader) Read(p []byte) (n int, err error) {
	if this.bytesPerSecs == 0 {
		return this.reader.Read(p)
	}

	lenP := len(p)
	index := 0

	curSec := time.Now().Truncate(time.Second)
	if this.lastSec.IsZero() || this.lastSec.Before(curSec) {
		this.lastSec = curSec
		this.lastSecRead = 0
	}

	for {
		sleep := false

		remainAllowedLen := this.bytesPerSecs - this.lastSecRead
		remainBufferLen := lenP - index

		if remainBufferLen > remainAllowedLen {
			remainBufferLen = remainAllowedLen
			sleep = true
		}

		n, err := this.reader.Read(p[index : index+remainBufferLen])

		index += n
		this.lastSecRead += n

		if err != nil {
			return index, err
		}

		if index == lenP {
			return index, nil
		}

		if sleep {
			this.lastSec = this.lastSec.Add(time.Second)
			this.lastSecRead = 0

			d := this.lastSec.Sub(time.Now())

			if d > 0 {
				time.Sleep(d)
			}
		}
	}
}

func NewThrottledReader(reader io.Reader, bytesPerSecs int) io.Reader {
	r := &throttledReader{
		reader:       reader,
		bytesPerSecs: bytesPerSecs,
	}

	return r
}
