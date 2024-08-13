package common

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestClone(t *testing.T) {
	require.Equal(t, 0, Clone(0))
	require.Equal(t, "a", Clone("a"))

	m := make(map[string]int)
	m["a"] = 1
	m["b"] = 2

	mCopy := Clone(m)

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

	sCopy := Clone(s)

	require.Equal(t, s, sCopy)

	sCopy.Address.Zip = 98765

	require.NotEqual(t, s, sCopy)
}
