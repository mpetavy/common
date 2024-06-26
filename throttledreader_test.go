package common

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
	"time"
)

func TestNewThrottledReader(t *testing.T) {
	SetTesting(t)

	data := []byte("123")
	bytesPerSecond := 1

	ba := make([]byte, len(data))
	throttledReader := NewThrottledReader(bytes.NewReader(data), bytesPerSecond)

	startTime := time.Now()
	_, err := ReadFully(throttledReader, ba)
	endTime := time.Now()

	needed := float64(endTime.Sub(startTime).Seconds())

	assert.NoError(t, err)

	assert.Equal(t, data, ba)

	expected := math.Ceil(float64(max(0, len(data)-bytesPerSecond)) / float64(bytesPerSecond))

	assert.GreaterOrEqual(t, needed, expected)
}
