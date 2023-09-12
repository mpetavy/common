package common

import (
	"io"
)

type memoryWriter struct {
	io.Writer

	msgs []string
}

func newMemoryWriter() *memoryWriter {
	return &memoryWriter{}
}

func (mw *memoryWriter) Write(msg []byte) (int, error) {
	if len(mw.msgs) >= *FlagLogCount {
		mw.msgs = mw.msgs[len(mw.msgs)-*FlagLogCount:]
	}

	mw.msgs = append(mw.msgs, string(msg))

	return len(msg), nil
}

func (mw *memoryWriter) Copy(w io.Writer) error {
	for _, msg := range mw.msgs {
		_, err := w.Write([]byte(msg))

		if err != nil {
			return err
		}
	}

	return nil
}

func (mw *memoryWriter) Clear() error {
	mw.msgs = nil

	return nil
}
