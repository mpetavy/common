package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRuntimeInfo(t *testing.T) {
	ri := GetRuntimeInfo(0)

	require.Equal(t, ri.Pack, "common")
	require.Equal(t, ri.File, "runtimeinfo_test.go")
	require.Equal(t, ri.Fn, "TestRuntimeInfo")
	require.Equal(t, ri.Line, 9)
}
