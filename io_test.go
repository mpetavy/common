package common

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
	"time"
)

const (
	timeout = time.Millisecond * 100
)

type freezer struct{}

func (f freezer) Read([]byte) (int, error) {
	time.Sleep(time.Hour)

	return 0, nil
}

func (f freezer) Write([]byte) (int, error) {
	time.Sleep(time.Hour)

	return 0, nil
}

func TestTimeoutReader(t *testing.T) {
	reader := NewTimeoutReader(&freezer{}, true, timeout)

	start := time.Now()

	ba := make([]byte, 1)
	_, err := reader.Read(ba)

	assert.Less(t, time.Since(start), timeout*2)
	assert.True(t, IsErrTimeout(err))
}

func TestTimeoutWriter(t *testing.T) {
	writer := NewTimeoutWriter(&freezer{}, true, timeout)

	start := time.Now()

	ba := make([]byte, 1)
	_, err := writer.Write(ba)

	assert.Less(t, time.Since(start), timeout*2)
	assert.True(t, IsErrTimeout(err))
}

type SingleByteWriter struct {
	Writer io.Writer
}

func (s SingleByteWriter) Write(p []byte) (n int, err error) {
	return s.Writer.Write(p[:1])
}

func TestWriteFully(t *testing.T) {
	data := []byte("123")
	buf := bytes.Buffer{}

	_, err := WriteFully(SingleByteWriter{&buf}, data)

	assert.NoError(t, err)
	assert.Equal(t, data, buf.Bytes())
}

type SingleByteReader struct {
	Reader io.Reader
}

func (s SingleByteReader) Read(p []byte) (n int, err error) {
	return s.Reader.Read(p[:1])
}

func TestReadFully(t *testing.T) {
	data := []byte("123")
	buf := make([]byte, len(data))

	_, err := ReadFully(SingleByteReader{bytes.NewReader(data)}, buf)

	assert.NoError(t, err)
	assert.Equal(t, data, buf)
}
