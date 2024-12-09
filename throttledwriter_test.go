package common

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"math"
	"testing"
	"time"
)

func TestNewThrottledWriter(t *testing.T) {
	data := []byte("123")
	bytesPerSecond := 1

	buf := &bytes.Buffer{}
	throttledWriter := NewThrottledWriter(buf, bytesPerSecond)

	startTime := time.Now()
	_, err := WriteFully(throttledWriter, data)
	endTime := time.Now()

	needed := float64(endTime.Sub(startTime).Seconds())

	require.NoError(t, err)

	require.Equal(t, data, buf.Bytes())

	expected := math.Ceil(float64(max(0, len(data)-bytesPerSecond)) / float64(bytesPerSecond))

	require.GreaterOrEqual(t, needed, expected)
}
