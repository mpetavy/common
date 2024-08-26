package common

import (
	"io"
	"sync/atomic"
)

type AutoCloser struct {
	Reader   io.Reader
	IsClosed atomic.Bool
}

func NewAutoCloser(reader io.Reader) *AutoCloser {
	return &AutoCloser{
		Reader:   reader,
		IsClosed: atomic.Bool{},
	}
}

func (ac *AutoCloser) Read(p []byte) (n int, err error) {
	n, err = ac.Reader.Read(p)
	if err == io.EOF {
		ac.IsClosed.Store(true)

		if closer, ok := ac.Reader.(io.ReadCloser); ok {
			Error(closer.Close())
		}
	}

	return n, err
}
