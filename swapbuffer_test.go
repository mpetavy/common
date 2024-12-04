package common

import (
	"bytes"
	"flag"
	"github.com/stretchr/testify/require"
	"io"
	"strconv"
	"testing"
)

func testSwapBuffer(t *testing.T, useCompression bool) {
	fl := flag.Lookup(FlagNameIoSwapBufferThreshold)
	err := fl.Value.Set("10")
	require.NoError(t, err)

	fl = flag.Lookup(FlagNameIoSwapBufferCompression)
	err = fl.Value.Set(strconv.FormatBool(useCompression))
	require.NoError(t, err)

	msg := RndBytes(100)

	sb := NewSwapBuffer()

	// test for "still to memory" write

	n, err := sb.Write(msg[:5])
	require.NoError(t, err)
	require.Equal(t, n, 5)

	// continue to force swap to disk

	n, err = sb.Write(msg[5:])
	require.NoError(t, err)
	require.Equal(t, n, 95)

	// check that is swapped to disk

	require.True(t, sb.OnDisk())

	// check for len is correct

	require.Equal(t, len(msg), sb.Len())

	// read bytes

	ba, err := io.ReadAll(sb)
	require.NoError(t, err)

	// check for reading back is correct

	require.Equal(t, msg, ba)

	var buf bytes.Buffer

	// create a reader

	r, err := sb.Reader()
	require.NoError(t, err)

	n64, err := io.Copy(&buf, r)
	require.NoError(t, err)

	// check for reading back is correct

	require.Equal(t, msg, buf.Bytes())

	// check for len is correct

	require.Equal(t, int64(len(msg)), n64)
	require.Equal(t, msg, ba)

	err = sb.Close()
	require.NoError(t, err)
}

func TestSwapBuffer(t *testing.T) {
	tests := []struct {
		name           string
		useCompression bool
	}{
		{
			name:           "With compression",
			useCompression: true,
		},
		{
			name:           "Without compression",
			useCompression: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testSwapBuffer(t, test.useCompression)
		})
	}
}
