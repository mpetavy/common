package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func checkError(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}

func c(s string) string {
	return strings.Replace(s, "'", "\"", -1)
}

func TestJason(t *testing.T) {
	j, err := NewJason(c("{ 'string':'s','int':1,'bool':true }"))
	checkError(t, err)

	b := j.IsString("string")
	require.True(t, b, "is a string")

	b = j.IsInt("string")
	require.False(t, b, "is a string")

	b = j.IsBool("string")
	require.False(t, b, "is a string")

	// ---

	b = j.IsString("int")
	require.False(t, b, "is a int")

	b = j.IsInt("int")
	require.True(t, b, "is a int")

	b = j.IsBool("int")
	require.False(t, b, "is a int")

	// ---

	b = j.IsString("bool")
	require.False(t, b, "is a bool")

	b = j.IsInt("bool")
	require.False(t, b, "is a bool")

	b = j.IsBool("bool")
	require.True(t, b, "is a bool")

	s, err := j.String("string")
	checkError(t, err)
	i, err := j.Int("int")
	checkError(t, err)
	b, err = j.Bool("bool")
	checkError(t, err)

	require.Equal(t, "s", s)
	require.Equal(t, 1, i)
	require.Equal(t, true, b)
}
