package common

import (
	"golang.org/x/exp/slices"
	"strings"
)

type MultiValueFlag []string

func (multiValueFLag *MultiValueFlag) String() string {
	if multiValueFLag == nil {
		return ""
	}

	return strings.Join(*multiValueFLag, ",")
}

func (multiValueFLag *MultiValueFlag) Set(s string) error {
	values := strings.Split(s, ",")

	for _, value := range values {
		if !slices.Contains(*multiValueFLag, value) {
			*multiValueFLag = append(*multiValueFLag, value)
		}
	}

	return nil
}
