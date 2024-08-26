package common

import (
	"flag"
)

var (
	SystemFlagNames []string
)

func systemFlagBool(name string, value bool, usage string) *bool {
	SystemFlagNames = append(SystemFlagNames, name)

	return flag.Bool(name, value, usage)
}

func systemFlagInt(name string, value int, usage string) *int {
	SystemFlagNames = append(SystemFlagNames, name)

	return flag.Int(name, value, usage)
}

func systemFlagInt64(name string, value int64, usage string) *int64 {
	SystemFlagNames = append(SystemFlagNames, name)

	return flag.Int64(name, value, usage)
}

func systemFlagString(name string, value string, usage string) *string {
	SystemFlagNames = append(SystemFlagNames, name)

	return flag.String(name, value, usage)
}
