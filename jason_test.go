package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
	j, err := NewJason(c("{ 'string':'s','int':'1','bool':'true' }"))
	checkError(t, err)

	b := j.IsString("string")
	assert.True(t, b, "is a string")

	b = j.IsInt("string")
	assert.False(t, b, "is a string")

	b = j.IsBool("string")
	assert.False(t, b, "is a string")

	// ---

	b = j.IsString("int")
	assert.False(t, b, "is a int")

	b = j.IsInt("int")
	assert.True(t, b, "is a int")

	b = j.IsBool("int")
	assert.False(t, b, "is a int")

	// ---

	b = j.IsString("bool")
	assert.False(t, b, "is a bool")

	b = j.IsInt("bool")
	assert.False(t, b, "is a bool")

	b = j.IsBool("bool")
	assert.True(t, b, "is a bool")

	s, err := j.String("string")
	checkError(t, err)
	i, err := j.Int("int")
	checkError(t, err)
	b, err = j.Bool("bool")
	checkError(t, err)

	assert.Equal(t, "s", s)
	assert.Equal(t, 1, i)
	assert.Equal(t, true, b)
}
