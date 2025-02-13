package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOptions(t *testing.T) {
	list := []string{"1", "2", "3"}

	options, err := NewOptions(list, []string{})
	require.NoError(t, err)
	require.True(t, options.Contains("1"))
	require.True(t, options.Contains("2"))
	require.True(t, options.Contains("2"))
	require.False(t, options.Contains("x"))

	options, err = NewOptions(list, []string{"x"})
	require.Error(t, err)

	options, err = NewOptions(list, []string{"1", "3"})
	require.NoError(t, err)
	require.True(t, options.Contains("1"))
	require.False(t, options.Contains("2"))
	require.True(t, options.Contains("3"))
	require.False(t, options.Contains("x"))

	options, err = NewOptions(list, []string{"-3"})
	require.NoError(t, err)
	require.True(t, options.Contains("1"))
	require.True(t, options.Contains("2"))
	require.False(t, options.Contains("3"))
	require.False(t, options.Contains("x"))

	options, err = NewOptions(list, []string{"-2"})
	require.NoError(t, err)
	require.True(t, options.Contains("1"))
	require.False(t, options.Contains("2"))
	require.True(t, options.Contains("3"))
	require.False(t, options.Contains("x"))
}
