package common

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewElement(t *testing.T) {
	e := NewElement("input")

	require.Equal(t, "<input/>", e.String())

	e.Text = "Hello World!"
	require.Equal(t, "<input>Hello World!</input>", e.String())

	e.AddAttr("type", "text")
	require.Equal(t, "<input type=\"text\">Hello World!</input>", e.String())

	e.AddAttr("readonly", "")
	require.Equal(t, "<input type=\"text\" readonly>Hello World!</input>", e.String())

	for i := 0; i < 3; i++ {
		e.AddElement(NewElement(fmt.Sprintf("elem%d", i)))
	}
	require.Equal(t, "<input type=\"text\" readonly><elem0/><elem1/><elem2/>Hello World!</input>", e.String())
}
