package common

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
	"time"
)

func TestNewThrottledWriter(t *testing.T) {
	InitTesting(t)

	data := []byte("123")
	bytesPerSecond := 1

	buf := &bytes.Buffer{}
	throttledWriter := NewThrottledWriter(buf, bytesPerSecond)

	startTime := time.Now()
	_, err := WriteFully(throttledWriter, data)
	endTime := time.Now()

	needed := float64(endTime.Sub(startTime).Seconds())

	assert.NoError(t, err)

	assert.Equal(t, data, buf.Bytes())

	expected := math.Ceil(float64(max(0, len(data)-bytesPerSecond)) / float64(bytesPerSecond))

	assert.GreaterOrEqual(t, needed, expected)
}
