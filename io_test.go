package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
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

func TestTimeoutReader1(t *testing.T) {
	reader := NewTimeoutReader(&freezer{}, true, time.Millisecond*100)

	ba := make([]byte, 1)
	_, err := reader.Read(ba)

	assert.True(t, IsErrTimeout(err))
}

func TestTimeoutWriter1(t *testing.T) {
	writer := NewTimeoutWriter(&freezer{}, true, time.Millisecond*100)

	ba := make([]byte, 1)
	_, err := writer.Write(ba)

	assert.True(t, IsErrTimeout(err))
}
