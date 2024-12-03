package common

import (
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func TestAutoCloser(t *testing.T) {
	msg := "Hello"

	file, err := CreateTempFile()
	require.NoError(t, err)

	err = os.WriteFile(file.Name(), []byte(msg), DefaultFileMode)
	require.NoError(t, err)

	r, err := os.OpenFile(file.Name(), os.O_RDONLY, os.ModePerm)
	require.NoError(t, err)

	ar := NewAutoCloser(r)
	ba, err := io.ReadAll(ar)
	require.NoError(t, err)

	require.Equal(t, msg, string(ba))

	require.True(t, ar.IsClosed.Load())

	require.NoError(t, os.Remove(file.Name()))
}
