package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCatch(t *testing.T) {
	err := Catch(func() {
		panic("panic")
	})

	assert.Equal(t, "panic", err.Error())
}
