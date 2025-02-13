package common

import (
	"fmt"
	"slices"
	"strings"
)

type Options struct {
	list []string
}

func NewOptions(allOptions []string, selectedOptions []string) (*Options, error) {
	options := &Options{
		list: slices.Clone(allOptions),
	}

	if len(selectedOptions) == 0 {
		return options, nil
	}

	includes := []string{}
	excludes := []string{}

	for _, option := range selectedOptions {
		isExclude := strings.HasPrefix(option, "-")
		if isExclude {
			option = strings.TrimPrefix(option, "-")
		}

		if !slices.Contains(allOptions, option) {
			return nil, fmt.Errorf("option %s not in list %v", option, allOptions)
		}

		if isExclude {
			excludes = append(excludes, option)
		} else {
			includes = append(includes, option)
		}
	}

	if len(excludes) == 0 {
		options = &Options{}
	} else {
		for _, option := range excludes {
			options.list = SliceRemove(options.list, option)
		}
	}

	for _, option := range includes {
		if !slices.Contains(options.list, option) {
			options.list = append(options.list, option)
		}
	}

	return options, nil
}

func (options *Options) Contains(option string) bool {
	return slices.Contains(options.list, option)
}
