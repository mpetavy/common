package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRuntimeInfo(t *testing.T) {
	ri := RuntimeInfo(0)

	assert.Equal(t, ri.Pack, "common")
	assert.Equal(t, ri.File, "runtimeinfo_test.go")
	assert.Equal(t, ri.Fn, "TestRuntimeInfo")
	assert.Equal(t, ri.Line, 9)
}
