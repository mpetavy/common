package common

import "strings"

type MultiValueFlag []string

func (multiValueFLag *MultiValueFlag) String() string {
	if multiValueFLag == nil {
		return ""
	}

	return strings.Join(*multiValueFLag, ",")
}

func (multiValueFLag *MultiValueFlag) Set(value string) error {
	splits := strings.Split(value, ",")
	*multiValueFLag = append(*multiValueFLag, splits...)

	return nil
}
