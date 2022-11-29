package common

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
	"time"
)

const (
	message = "Hello world"
)

func readerTest(t *testing.T, immediateContext bool) (bytes.Buffer, int64, error) {
	pr, pw := io.Pipe()

	go func() {
		// close the writer, so the reader knows there's no more data
		defer pw.Close()

		time.Sleep(time.Second)

		pw.Write([]byte(message))
	}()

	buf := bytes.Buffer{}

	reader := NewTimeoutReader(pr, immediateContext,
		func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), time.Duration(500*time.Millisecond))
		})

	n, err := io.Copy(&buf, reader)

	return buf, n, err
}

func TestTimeoutReader(t *testing.T) {
	buf, n, err := readerTest(t, false)

	assert.Nil(t, err)
	assert.Equal(t, int(n), buf.Len())
	assert.Equal(t, message, string(buf.Bytes()))
}

func TestTimeoutReaderError(t *testing.T) {
	_, _, err := readerTest(t, true)

	assert.NotNil(t, err)
	assert.Equal(t, true, IsErrTimeout(err))
}

type writer struct {
	ImmediateContext bool
	W                io.Writer
	Sleep            time.Duration
}

func (w writer) Write(p []byte) (int, error) {
	n := 0
	if w.ImmediateContext {
		var err error

		n, err = w.W.Write(p[:1])
		if err != nil {
			return n, err
		}

		p = p[1:]

		time.Sleep(w.Sleep)
	}

	n1, err := w.W.Write(p)

	return n + n1, err
}

func writerTest(t *testing.T, immediateContext bool) (bytes.Buffer, int, error) {
	buf := bytes.Buffer{}
	w := &writer{
		ImmediateContext: immediateContext,
		W:                &buf,
		Sleep:            time.Second,
	}

	writer := NewTimeoutWriter(w, immediateContext,
		func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), time.Duration(500*time.Millisecond))
		})

	n, err := writer.Write([]byte(message))

	return buf, n, err
}

func TestTimeoutWriter(t *testing.T) {
	buf, n, err := writerTest(t, false)

	assert.Nil(t, err)
	assert.Equal(t, int(n), buf.Len())
	assert.Equal(t, message, string(buf.Bytes()))
}

func TestTimeoutWriterError(t *testing.T) {
	_, _, err := writerTest(t, true)

	assert.NotNil(t, err)
	assert.Equal(t, true, IsErrTimeout(err))
}
