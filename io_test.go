package common

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"io"
	"strings"
	"testing"
	"time"
)

const (
	message = "Hello world"
	timeout = time.Millisecond * 500
)

func readerTest(reader io.Reader, initialContext bool, timeout time.Duration) ([]byte, int64, error) {
	if !initialContext {
		time.Sleep(timeout)
	}

	buf := bytes.Buffer{}
	reader = NewTimeoutReader(reader, initialContext,
		func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), timeout)
		})

	n, err := io.Copy(&buf, reader)

	return buf.Bytes(), n, err
}

func TestTimeoutReader(t *testing.T) {
	tests := []struct {
		name           string
		initialContext bool
		reader         io.Reader
		isErrTimeout   bool
		message        string
	}{
		{
			name:           "0",
			initialContext: true,
			reader:         strings.NewReader(message),
			isErrTimeout:   false,
			message:        message,
		},
		{
			name:           "1",
			initialContext: false,
			reader:         strings.NewReader(message),
			isErrTimeout:   false,
			message:        message,
		},
		{
			name:           "2",
			initialContext: true,
			reader:         NewRandomReader(),
			isErrTimeout:   true,
			message:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()

			ba, n, err := readerTest(tt.reader, tt.initialContext, timeout)

			if tt.isErrTimeout {
				assert.True(t, IsErrTimeout(err))
			} else {
				assert.Equal(t, nil, err)
			}

			if tt.message != "" {
				assert.Equal(t, int(n), len(ba))
				assert.Equal(t, tt.message, string(ba))
			}

			if !tt.initialContext {
				assert.Greater(t, timeout*4, time.Since(start))
			} else {
				assert.Greater(t, timeout*2, time.Since(start))
			}
		})
	}
}

func writerTest(reader io.Reader, initialContext bool, timeout time.Duration) ([]byte, int64, error) {
	if !initialContext {
		time.Sleep(timeout)
	}

	buf := &bytes.Buffer{}

	var writer io.Writer
	writer = buf

	writer = NewTimeoutWriter(writer, initialContext,
		func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), timeout)
		})

	n, err := io.Copy(writer, reader)

	return buf.Bytes(), n, err
}

func TestTimeoutWriter(t *testing.T) {
	tests := []struct {
		name           string
		initialContext bool
		reader         io.Reader
		isErrTimeout   bool
		message        string
	}{
		{
			name:           "0",
			initialContext: true,
			reader:         strings.NewReader(message),
			isErrTimeout:   false,
			message:        message,
		},
		{
			name:           "1",
			initialContext: false,
			reader:         strings.NewReader(message),
			isErrTimeout:   false,
			message:        message,
		},
		{
			name:           "2",
			initialContext: true,
			reader:         NewRandomReader(),
			isErrTimeout:   true,
			message:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()

			ba, n, err := writerTest(tt.reader, tt.initialContext, timeout)

			if tt.isErrTimeout {
				assert.True(t, IsErrTimeout(err))
			} else {
				assert.Equal(t, nil, err)
			}

			if tt.message != "" {
				assert.Equal(t, int(n), len(ba))
				assert.Equal(t, tt.message, string(ba))
			}

			if !tt.initialContext {
				assert.Greater(t, timeout*4, time.Since(start))
			} else {
				assert.Greater(t, timeout*2, time.Since(start))
			}
		})
	}
}
