package common

import (
	"fmt"
	"html"
	"strings"
)

type Element struct {
	Name      string
	Text      string
	PlainText bool
	Attrs     *OrderedMap[string, string]
	Elements  []*Element
}

func NewElement(name string) *Element {
	return &Element{
		Name:  name,
		Attrs: NewOrderedMap[string, string](),
	}
}

func (e *Element) String() string {
	sb := strings.Builder{}

	if e.IsTextOnly() {
		if e.Text != "" {
			if e.PlainText {
				sb.WriteString(e.Text)
			} else {
				sb.WriteString(html.EscapeString(e.Text))
			}
		}

		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("<%s", e.Name))

	if e.Attrs.Len() == 0 && e.Text == "" {
		sb.WriteString("/>")

		return sb.String()
	}

	e.Attrs.Range(func(k string, v string) {
		if sb.Len() > 0 {
			sb.WriteString(" ")
		}

		if v == "" {
			sb.WriteString(k)
		} else {
			sb.WriteString(fmt.Sprintf("%s=\"%s\"", k, html.EscapeString(v)))
		}
	})

	sb.WriteString(">")

	for _, element := range e.Elements {
		sb.WriteString(element.String())
	}

	if e.Text != "" {
		if e.PlainText {
			sb.WriteString(e.Text)
		} else {
			sb.WriteString(html.EscapeString(e.Text))
		}
	}

	sb.WriteString(fmt.Sprintf("</%s>", e.Name))

	return sb.String()
}

func (e *Element) IsTextOnly() bool {
	return e.Name == ""
}

func (e *Element) AddElement(element *Element) *Element {
	e.Elements = append(e.Elements, element)

	return element
}

func (e *Element) AddElementName(name string) *Element {
	element := NewElement(name)

	e.AddElement(element)

	return element
}

func (e *Element) RemoveElement(element *Element) *Element {
	e.Elements = append(e.Elements, element)

	return e
}

func (e *Element) AddAttr(name string, value string) *Element {
	e.Attrs.Add(name, value)

	return e
}

func (e *Element) RemoveAttr(name string) *Element {
	e.Attrs.Remove(name)

	return e
}
