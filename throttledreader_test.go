package common

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
	"time"
)

func TestNewThrottledReader(t *testing.T) {
	data := []byte("123")
	bytesPerSecond := 1

	ba := make([]byte, len(data))
	throttledReader := NewThrottledReader(bytes.NewReader(data), bytesPerSecond)

	startTime := time.Now()
	_, err := ReadFully(throttledReader, ba)
	endTime := time.Now()

	needed := float64(endTime.Sub(startTime).Seconds())

	require.NoError(t, err)

	require.Equal(t, data, ba)

	expected := math.Ceil(float64(max(0, len(data)-bytesPerSecond)) / float64(bytesPerSecond))

	require.GreaterOrEqual(t, needed, expected)
}
