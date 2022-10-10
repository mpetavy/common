package common

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewElement(t *testing.T) {
	e := NewElement("input")

	assert.Equal(t, "<input/>", e.String())

	e.Text = "Hello World!"
	assert.Equal(t, "<input>Hello World!</input>", e.String())

	e.AddAttr("type", "text")
	assert.Equal(t, "<input type=\"text\">Hello World!</input>", e.String())

	e.AddAttr("readonly", "")
	assert.Equal(t, "<input type=\"text\" readonly>Hello World!</input>", e.String())

	for i := 0; i < 3; i++ {
		e.AddElement(NewElement(fmt.Sprintf("elem%d", i)))
	}
	assert.Equal(t, "<input type=\"text\" readonly><elem0/><elem1/><elem2/>Hello World!</input>", e.String())
}
