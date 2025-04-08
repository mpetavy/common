package common

import (
	"fmt"
	"slices"
	"strings"
)

type Options struct {
	All      []string
	Includes []string
	Excludes []string
}

func NewOptions(allOptions []string, selectedOptions []string) (*Options, error) {
	options := &Options{
		All: slices.Clone(allOptions),
	}

	if len(selectedOptions) == 0 {
		return options, nil
	}

	for _, option := range selectedOptions {
		isExclude := strings.HasPrefix(option, "-")
		if isExclude {
			option = strings.TrimPrefix(option, "-")
		}

		if len(allOptions) > 0 && !slices.Contains(allOptions, option) {
			return nil, fmt.Errorf("option %s not in list %v", option, allOptions)
		}

		if isExclude {
			options.Excludes = append(options.Excludes, option)
		} else {
			options.Includes = append(options.Includes, option)
		}
	}

	return options, nil
}

func (options *Options) IsValid(option string) bool {
	if len(options.All) > 0 && !slices.Contains(options.All, option) {
		return false
	}

	if options.Includes == nil {
		return !slices.ContainsFunc(options.Excludes, func(s string) bool {
			b, err := EqualsWildcard(option, s)

			return b && err == nil
		})
	} else {
		return slices.ContainsFunc(options.Includes, func(s string) bool {
			b, err := EqualsWildcard(option, s)

			return b && err == nil
		})
	}
}
