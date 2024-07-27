package common

import (
	"fmt"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"html"
	"strings"
)

type Element struct {
	Name      string
	Text      string
	PlainText bool
	Attrs     *orderedmap.OrderedMap[string, string]
	Elements  []*Element
}

func NewElement(name string) *Element {
	return &Element{
		Name:  name,
		Attrs: orderedmap.New[string, string](),
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

	for pair := e.Attrs.Oldest(); pair != nil; pair = pair.Next() {
		if sb.Len() > 0 {
			sb.WriteString(" ")
		}

		if pair.Value == "" {
			sb.WriteString(pair.Key)
		} else {
			sb.WriteString(fmt.Sprintf("%s=\"%s\"", pair.Key, html.EscapeString(pair.Value)))
		}
	}

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
	e.Attrs.Set(name, value)

	return e
}

func (e *Element) RemoveAttr(name string) *Element {
	e.Attrs.Delete(name)

	return e
}
