package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestClone(t *testing.T) {
	x, err := Clone(0)
	require.NoError(t, err)
	require.Equal(t, 0, x)

	y, err := Clone("a")
	require.NoError(t, err)
	require.Equal(t, "a", y)

	m := make(map[string]int)
	m["a"] = 1
	m["b"] = 2

	mCopy, err := Clone(m)
	require.NoError(t, err)

	require.Equal(t, m, mCopy)

	mCopy["c"] = 3

	require.NotEqual(t, m, mCopy)

	type address struct {
		City string
		Zip  int
	}

	type patient struct {
		Name    string
		Address address
	}

	s := patient{
		Name: "test",
		Address: address{
			City: "New York",
			Zip:  12345,
		},
	}

	sCopy, err := Clone(s)
	require.NoError(t, err)

	require.Equal(t, s, sCopy)

	sCopy.Address.Zip = 98765

	require.NotEqual(t, s, sCopy)
}
