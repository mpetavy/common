package common

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

const (
	FlagNameIoSwapThreshold = "io.swap.threshold"
)

var (
	FlagIoSwapThreshold = SystemFlagInt(FlagNameIoSwapThreshold, 10*1024*1024, "byte count threshold to store to file")
)

type SwapBuffer struct {
	io.ReadWriteCloser
	buf        bytes.Buffer
	file       *os.File
	inReadMode bool
	isClosed   bool
	written    int
}

func NewSwapBuffer() *SwapBuffer {
	return &SwapBuffer{}
}

func (sb *SwapBuffer) OnDisk() bool {
	return sb.file != nil
}

func (sb *SwapBuffer) Write(p []byte) (int, error) {
	if sb.isClosed {
		return 0, os.ErrInvalid
	}

	if sb.inReadMode {
		return 0, fmt.Errorf("already in READ mode")
	}

	n := len(p)

	if !sb.OnDisk() {
		if sb.written+len(p) > *FlagIoSwapThreshold {
			f, err := CreateTempFile()
			if err != nil {
				return 0, err
			}

			sb.file, err = os.OpenFile(f.Name(), os.O_CREATE|os.O_APPEND|os.O_RDWR, DefaultFileMode)
			if err != nil {
				return 0, err
			}

			_, err = sb.file.Write(sb.buf.Bytes())
			if err != nil {
				return 0, err
			}

			sb.buf.Reset()
		}
	}

	if sb.OnDisk() {
		var err error

		n, err = sb.file.Write(p)
		if err != nil {
			return 0, err
		}
	} else {
		var err error

		n, err = sb.buf.Write(p)
		if err != nil {
			return 0, err
		}
	}

	sb.written += n

	return n, nil
}

func (sb *SwapBuffer) WriteString(s string) (int, error) {
	return sb.Write([]byte(s))
}

func (sb *SwapBuffer) Read(p []byte) (int, error) {
	if sb.isClosed {
		return 0, os.ErrInvalid
	}

	if !sb.inReadMode {
		sb.inReadMode = true

		if sb.OnDisk() {
			_, err := sb.file.Seek(0, io.SeekStart)
			if err != nil {
				return 0, err
			}
		}
	}

	if sb.OnDisk() {
		n, err := sb.file.Read(p)
		if err != nil {
			return 0, err
		}

		return n, nil
	} else {
		n, err := sb.buf.Read(p)
		if err != nil {
			return 0, err
		}

		return n, nil
	}
}

func (sb *SwapBuffer) Close() error {
	if sb.OnDisk() {
		err := sb.file.Close()
		if err != nil {
			return err
		}

		err = os.Remove(sb.file.Name())
		if err != nil {
			return err
		}

		sb.isClosed = true
	}

	return nil
}

func (sb *SwapBuffer) Reader() (io.ReadCloser, error) {
	if sb.isClosed {
		return nil, os.ErrInvalid
	}

	sb.inReadMode = true

	if sb.OnDisk() {
		_, err := sb.file.Seek(0, io.SeekStart)
		if err != nil {
			return nil, err
		}

		return sb.file, nil
	}

	return io.NopCloser(bytes.NewReader(sb.buf.Bytes())), nil
}

func (sb *SwapBuffer) Len() int {
	return sb.written
}
