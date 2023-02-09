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

func readerTest(initialContext bool) (bytes.Buffer, int64, error) {
	pr, pw := io.Pipe()

	go func() {
		// close the writer, so the reader knows there's no more data
		defer func() {
			Error(pw.Close())
		}()

		time.Sleep(time.Second)

		pw.Write([]byte(message))
	}()

	buf := bytes.Buffer{}

	reader := NewTimeoutReader(pr, initialContext,
		func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), time.Duration(500*time.Millisecond))
		})

	n, err := io.Copy(&buf, reader)

	return buf, n, err
}

func TestTimeoutReader(t *testing.T) {
	buf, n, err := readerTest(false)

	assert.Nil(t, err)
	assert.Equal(t, int(n), buf.Len())
	assert.Equal(t, message, string(buf.Bytes()))
}

func TestTimeoutReaderError(t *testing.T) {
	_, _, err := readerTest(true)

	assert.NotNil(t, err)
	assert.Equal(t, true, IsErrTimeout(err))
}

type writer struct {
	initialContext bool
	w              io.Writer
	sleep          time.Duration
}

func (w writer) Write(p []byte) (int, error) {
	n := 0
	if w.initialContext {
		var err error

		n, err = w.w.Write(p[:1])
		if err != nil {
			return n, err
		}

		p = p[1:]

		time.Sleep(w.sleep)
	}

	n1, err := w.w.Write(p)

	return n + n1, err
}

func writerTest(initialContext bool) (bytes.Buffer, int, error) {
	buf := bytes.Buffer{}
	w := &writer{
		initialContext: initialContext,
		w:              &buf,
		sleep:          time.Second,
	}

	writer := NewTimeoutWriter(w, initialContext,
		func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), time.Duration(500*time.Millisecond))
		})

	n, err := writer.Write([]byte(message))

	return buf, n, err
}

func TestTimeoutWriter(t *testing.T) {
	buf, n, err := writerTest(false)

	assert.Nil(t, err)
	assert.Equal(t, int(n), buf.Len())
	assert.Equal(t, message, string(buf.Bytes()))
}

func TestTimeoutWriterError(t *testing.T) {
	_, _, err := writerTest(true)

	assert.NotNil(t, err)
	assert.Equal(t, true, IsErrTimeout(err))
}
