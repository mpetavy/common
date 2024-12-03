package common

import (
	"io"
	"sync/atomic"
)

type AutoCloser struct {
	Reader   io.Reader
	IsClosed atomic.Bool
	err      error
}

func NewAutoCloser(reader io.Reader) *AutoCloser {
	return &AutoCloser{
		Reader:   reader,
		IsClosed: atomic.Bool{},
	}
}

func (ac *AutoCloser) Read(p []byte) (int, error) {
	if ac.err != nil {
		return 0, ac.err
	}

	var n int

	n, ac.err = ac.Reader.Read(p)
	if ac.err == io.EOF {
		return n, ac.Close()
	}

	return n, ac.err
}

func (ac *AutoCloser) Close() error {
	if ac.IsClosed.Load() {
		return nil
	}

	ac.IsClosed.Store(true)

	if closer, ok := ac.Reader.(io.Closer); ok {
		return closer.Close()
	}

	return nil
}
