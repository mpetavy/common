package common

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
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

func TestCreateTempFile(t *testing.T) {
	tempFile, err := CreateTempFile()
	require.NoError(t, err)

	tempDir, err := CreateTempDir()
	require.NoError(t, err)

	tempFile, err = CreateTempFile(tempDir)
	require.NoError(t, err)

	require.Equal(t, tempDir, filepath.Dir(tempFile.Name()))
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
	dir, err := CreateTempDir()
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.RemoveAll(dir))
	}()

	filename := filepath.Join(dir, "common.log")

	for i := range 10 {
		err := FileBackup(filename)
		require.NoError(t, err)

		f, err := os.Create(filename)
		require.NoError(t, err)

		_, err = fmt.Fprintf(f, "%d\n", i)
		require.NoError(t, err)

		err = f.Close()
		require.NoError(t, err)
	}

	files, err := ListFiles(filename+"*", false)
	require.Equal(t, len(files), *FlagIoFileBackups+1)
}

func TestListFiles(t *testing.T) {
	dir, err := CreateTempDir()
	require.NoError(t, err)

	defer func() {
		require.NoError(t, os.RemoveAll(dir))
	}()

	subdir := filepath.Join(dir, "subdir")

	err = os.MkdirAll(subdir, DefaultDirMode)
	require.NoError(t, err)

	files := []string{
		filepath.Join(dir, "file0.txt"),
		filepath.Join(dir, "file1.txt"),
		filepath.Join(dir, "file1.txt.backup"),
		filepath.Join(dir, "other.ini"),
		filepath.Join(subdir, "file0.txt"),
		filepath.Join(subdir, "file1.txt"),
		filepath.Join(subdir, "other.ini"),
	}

	for _, file := range files {
		f, err := os.Create(file)
		require.NoError(t, err)

		err = f.Close()
		require.NoError(t, err)
	}

	found, err := ListFiles(filepath.Join(dir, "*.xxx"), false)
	require.NoError(t, err)
	require.Equal(t, 0, len(found))

	found, err = ListFiles(filepath.Join(dir, "*.xxx"), true)
	require.NoError(t, err)
	require.Equal(t, 0, len(found))

	found, err = ListFiles(filepath.Join(dir, "*.txt"), false)
	require.NoError(t, err)
	require.Equal(t, 2, len(found))

	found, err = ListFiles(filepath.Join(dir, "*.txt"), false)
	require.NoError(t, err)
	require.Equal(t, 2, len(found))

	found, err = ListFiles(filepath.Join(dir, "*.txt"), true)
	require.NoError(t, err)
	require.Equal(t, 4, len(found))

	found, err = ListFiles(filepath.Join(dir, "*.ini"), true)
	require.NoError(t, err)
	require.Equal(t, 2, len(found))
}

func TestSplitFilemask(t *testing.T) {
	type args struct {
		filemask string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 string
	}{
		{
			name:  "0",
			args:  args{"asdf"},
			want:  CleanPath("."),
			want1: "asdf",
		},
		{
			name:  "1",
			args:  args{"."},
			want:  CleanPath("."),
			want1: "*",
		},
		{
			name:  "2",
			args:  args{"*.txt"},
			want:  CleanPath("."),
			want1: "*.txt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := SplitFilemask(tt.args.filemask)
			if got != tt.want {
				t.Errorf("SplitFilemask() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("SplitFilemask() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
