package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCatch(t *testing.T) {
	err := Catch(func() error {
		panic("panic")
	})

	require.Equal(t, "panic", err.Error())
}
