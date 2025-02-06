package common

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

const (
	FlagNameIoSwapBufferThreshold   = "io.swapbuffer.threshold"
	FlagNameIoSwapBufferCompression = "io.swapbuffer.compression"
)

var (
	FlagIoSwapBufferThreshold   = SystemFlagInt(FlagNameIoSwapBufferThreshold, 1*1024*1024, "bytes count threshold to store to file")
	FlagIoSwapBufferCompression = SystemFlagBool(FlagNameIoSwapBufferCompression, true, "use compression with SwapBuffer")
)

type SwapBuffer struct {
	io.ReadWriteCloser
	buf        bytes.Buffer
	file       *os.File
	gzipWriter *gzip.Writer
	gzipReader *gzip.Reader
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
		if sb.written+len(p) > *FlagIoSwapBufferThreshold {
			f, err := CreateTempFile()
			if err != nil {
				return 0, err
			}

			sb.file, err = os.OpenFile(f, os.O_CREATE|os.O_APPEND|os.O_RDWR, DefaultFileMode)
			if err != nil {
				return 0, err
			}

			if *FlagIoSwapBufferCompression {
				sb.gzipWriter = gzip.NewWriter(sb.file)
			}

			sb.written = 0

			n, err = sb.writeBytes(sb.buf.Bytes())
			if err != nil {
				return 0, err
			}

			sb.buf.Reset()
		}
	}

	n, err := sb.writeBytes(p)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (sb *SwapBuffer) writeBytes(p []byte) (int, error) {
	if sb.OnDisk() {
		if *FlagIoSwapBufferCompression {
			n, err := sb.gzipWriter.Write(p)
			if err != nil {
				return 0, err
			}

			err = sb.gzipWriter.Flush()
			if err != nil {
				return 0, err
			}

			sb.written += n

			return n, nil
		} else {
			n, err := sb.file.Write(p)
			if err != nil {
				return 0, err
			}

			sb.written += n

			return n, err
		}
	} else {
		var err error

		n, err := sb.buf.Write(p)
		if err != nil {
			return 0, err
		}

		sb.written += n

		return n, nil
	}
}

func (sb *SwapBuffer) readBytes(p []byte) (int, error) {
	if *FlagIoSwapBufferCompression {
		return sb.gzipReader.Read(p)
	} else {
		return sb.file.Read(p)
	}
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

			if *FlagIoSwapBufferCompression {
				sb.gzipReader, err = gzip.NewReader(sb.file)
				if err != nil {
					return 0, err
				}
			}
		}
	}

	if sb.OnDisk() {
		if *FlagIoSwapBufferCompression {
			n, err := sb.gzipReader.Read(p)
			if err == io.ErrUnexpectedEOF {
				err = io.EOF
			}

			if err != nil {
				return 0, err
			}

			return n, nil
		} else {
			return sb.file.Read(p)
		}
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

func (sb *SwapBuffer) Len() int {
	return sb.written
}

func (sb *SwapBuffer) CompressedLen() (int, error) {
	if sb.OnDisk() {
		l, err := FileSize(sb.file.Name())
		if Error(err) {
			return 0, err
		}

		return int(l), nil
	}

	return sb.Len(), nil
}
