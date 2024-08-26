package common

import (
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestHybridWriter(t *testing.T) {
	*FlagIoBufferThreshold = 10

	msg, err := RndBytes(100)
	require.NoError(t, err)

	hb := NewHybridBuffer()

	n, err := hb.Write(msg)
	require.NoError(t, err)

	require.Equal(t, len(msg), n)

	r, err := hb.BytesReader()
	require.NoError(t, err)

	ba, err := io.ReadAll(r)
	require.NoError(t, err)

	require.Equal(t, msg, ba)

	autocloser, ok := r.(*AutoCloser)
	require.True(t, ok)

	require.True(t, autocloser.IsClosed.Load())
}
