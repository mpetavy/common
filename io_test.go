package common

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"os"
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

	require.Less(t, time.Since(start), timeout*2)
	require.True(t, IsErrTimeout(err))
}

func TestTimeoutWriter(t *testing.T) {
	writer := NewTimeoutWriter(&freezer{}, true, timeout)

	start := time.Now()

	ba := make([]byte, 1)
	_, err := writer.Write(ba)

	require.Less(t, time.Since(start), timeout*2)
	require.True(t, IsErrTimeout(err))
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

	require.NoError(t, err)
	require.Equal(t, data, buf.Bytes())
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

	require.NoError(t, err)
	require.Equal(t, data, buf)
}

func TestFileMode(t *testing.T) {
	f, err := CreateTempFile()
	require.NoError(t, err)

	err = os.Remove(f.Name())
	require.NoError(t, err)

	f, err = os.OpenFile(f.Name(), os.O_CREATE|os.O_TRUNC|os.O_RDWR, ReadOnlyFileMode)
	require.NoError(t, err)

	err = f.Close()
	require.NoError(t, err)

	err = os.Chmod(f.Name(), DefaultFileMode)
	require.NoError(t, err)

	err = os.Remove(f.Name())
	require.NoError(t, err)
}

func TestFileBackup(t *testing.T) {
	f, err := CreateTempFile()
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		f, err := os.Create(fmt.Sprintf("%s.%d", f.Name(), i+1))
		require.NoError(t, err)

		err = f.Close()
		require.NoError(t, err)
	}

	for i := 0; i < 5; i++ {
		err := FileBackup(f.Name())
		require.NoError(t, err)
	}

	files, err := ListFiles(f.Name()+"*", false)
	require.Equal(t, len(files), *FlagIoFileBackups+1)

	for _, file := range files {
		err = FileDelete(file)
		require.NoError(t, err)
	}
}
