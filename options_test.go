package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOptionsWithConcretes(t *testing.T) {
	list := []string{"1", "2", "3"}

	options, err := NewOptions(list, []string{})
	require.NoError(t, err)
	require.True(t, options.IsValid("1"))
	require.True(t, options.IsValid("2"))
	require.True(t, options.IsValid("2"))
	require.False(t, options.IsValid("x"))

	options, err = NewOptions(list, []string{"x"})
	require.Error(t, err)

	options, err = NewOptions(list, []string{"1", "3"})
	require.NoError(t, err)
	require.True(t, options.IsValid("1"))
	require.False(t, options.IsValid("2"))
	require.True(t, options.IsValid("3"))
	require.False(t, options.IsValid("x"))

	options, err = NewOptions(list, []string{"-3"})
	require.NoError(t, err)
	require.True(t, options.IsValid("1"))
	require.True(t, options.IsValid("2"))
	require.False(t, options.IsValid("3"))
	require.False(t, options.IsValid("x"))

	options, err = NewOptions(list, []string{"-2"})
	require.NoError(t, err)
	require.True(t, options.IsValid("1"))
	require.False(t, options.IsValid("2"))
	require.True(t, options.IsValid("3"))
	require.False(t, options.IsValid("x"))
}

func TestOptionsEmpty(t *testing.T) {
	options, err := NewOptions(nil, nil)
	require.NoError(t, err)

	require.True(t, options.IsValid("*.go"))
	require.True(t, options.IsValid("*.html"))
	require.True(t, options.IsValid("*.tmp"))
	require.True(t, options.IsValid("*.out"))
}

func TestOptionsWithoutConcretes(t *testing.T) {
	filemask := "-*.tmp,-*.out"

	options, err := NewOptions(nil, Split(filemask, ","))
	require.NoError(t, err)

	require.True(t, options.IsValid("*.go"))
	require.True(t, options.IsValid("*.html"))
	require.False(t, options.IsValid("*.tmp"))
	require.False(t, options.IsValid("*.out"))

	filemask = "*.html,-*.tmp,-*.out"

	options, err = NewOptions(nil, Split(filemask, ","))
	require.NoError(t, err)

	require.False(t, options.IsValid("*.go"))
	require.True(t, options.IsValid("*.html"))
	require.False(t, options.IsValid("*.tmp"))
	require.False(t, options.IsValid("*.out"))
}
