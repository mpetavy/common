package common

import (
	"encoding/json"
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

func TestJsonReformat(t *testing.T) {
	type person struct {
		Name string `json:"Name"`
		Age  int    `json:"Age"`
	}

	p := &person{
		Name: "bob",
		Age:  17,
	}

	ba0, err := json.Marshal(p)
	require.NoError(t, err)

	ba1, err := json.MarshalIndent(p, "", "    ")
	require.NoError(t, err)

	ba0, err = ReformatJson(ba0)
	require.NoError(t, err)

	ba1, err = ReformatJson(ba1)
	require.NoError(t, err)

	require.Equal(t, string(ba0), string(ba1))
}

func TestJsonFieldName(t *testing.T) {
	type person struct {
		Name string `json:"PersonName"`
		Age  int    `json:"PersonAge"`
	}

	p := &person{}

	n, err := JsonFieldName(p, "Name")
	require.NoError(t, err)
	require.Equal(t, "PersonName", n)

	n, err = JsonFieldName(p, "Age")
	require.NoError(t, err)
	require.Equal(t, "PersonAge", n)

	require.Equal(t, "Name", JsonDefaultName("name"))
}

func TestRemoveJsonComments(t *testing.T) {
	input := `{
  "name": "Alice", // this is a comment
  "url": "http://example.com", // keep this
  "note": "she said: \"// not a comment\"" // still not a comment
  "array": [
    {
      "elem0": "0" // remove this
      // ${script:getDurationInSeconds:${devicelog.ts},${job.videoStart}}
    },
	{"elem1": "1"},
	//{"elem2": "2"}, 
	//{"elem3": "3"},
  ]
}
// full line comment
`
	expected := `{
  "name": "Alice", 
  "url": "http://example.com", 
  "note": "she said: \"// not a comment\"" 
  "array": [
    {
      "elem0": "0" 
    },
	{"elem1": "1"}
  ]
}
`
	output, err := RemoveJsonComments([]byte(input))
	require.NoError(t, err)

	require.Equal(t, expected, string(output))
}
