package common

import (
	"bytes"
	"io"
	"os"
	"sync/atomic"
)

const (
	FlagNameIoBufferThreshold = "io.hybrid.threshold"
)

var (
	FlagIoBufferThreshold = SystemFlagInt(FlagNameIoBufferThreshold, 10*1024*1024, "byte threshold to store to file")
)

type HybridBuffer struct {
	io.ReadWriteCloser
	buf    *bytes.Buffer
	file   *os.File
	count  int
	inRead atomic.Bool
}

func NewHybridBuffer() *HybridBuffer {
	return &HybridBuffer{
		buf: &bytes.Buffer{},
	}
}

func (hb *HybridBuffer) Write(p []byte) (int, error) {
	if hb.inRead.Load() {
		_, err := hb.file.Seek(0, io.SeekEnd)
		if Error(err) {
			return 0, err
		}

		hb.inRead.Store(false)
	}

	if hb.file == nil && hb.count+len(p) > *FlagIoBufferThreshold {

		tempFile, err := CreateTempFile()
		if Error(err) {
			return 0, err
		}

		tempFile, err = os.OpenFile(tempFile.Name(), os.O_APPEND|os.O_RDWR, DefaultFileMode)
		if Error(err) {
			return 0, err
		}

		n, err := hb.buf.WriteTo(tempFile)
		if Error(err) {
			return int(n), err
		}

		hb.file = tempFile
		hb.buf = nil
	}

	hb.count += len(p)

	if hb.file != nil {
		return hb.file.Write(p)
	}

	return hb.buf.Write(p)
}

func (hb *HybridBuffer) Close() error {
	if hb.file != nil {
		err := hb.file.Close()
		if Error(err) {
			return err
		}

		err = os.Remove(hb.file.Name())
		if Error(err) {
			return err
		}
	}

	return nil
}

func (hb *HybridBuffer) BytesReader() (io.Reader, error) {
	hb.inRead.Store(true)

	if hb.file != nil {
		_, err := hb.file.Seek(0, io.SeekStart)
		if Error(err) {
			return nil, err
		}

		return NewAutoCloser(hb.file), nil
	}

	return NewAutoCloser(bytes.NewReader(hb.buf.Bytes())), nil
}

func (hb *HybridBuffer) Len() int {
	return hb.count
}
