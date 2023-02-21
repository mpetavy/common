package common

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	timeout = time.Millisecond * 100
)

func TestCtxReader(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()

	reader := NewCtxReader(ctx, NewRandomReader())

	ba := make([]byte, 1024)

	var err error
	for err == nil {
		_, err = reader.Read(ba)
	}

	assert.Less(t, time.Since(start), timeout*2)
	assert.True(t, IsErrTimeout(err))
}

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
