package common

import (
	"flag"
	"fmt"
)

var (
	SystemFlagNames []string
)

type ErrFlagNotDefined struct {
	Name string
}

func (e *ErrFlagNotDefined) Error() string {
	return fmt.Sprintf("Flags must be defined: %s", e.Name)
}

func SystemFlagBool(name string, value bool, usage string) *bool {
	SystemFlagNames = append(SystemFlagNames, name)

	return flag.Bool(name, value, usage)
}

func SystemFlagInt(name string, value int, usage string) *int {
	SystemFlagNames = append(SystemFlagNames, name)

	return flag.Int(name, value, usage)
}

func SystemFlagInt64(name string, value int64, usage string) *int64 {
	SystemFlagNames = append(SystemFlagNames, name)

	return flag.Int64(name, value, usage)
}

func SystemFlagString(name string, value string, usage string) *string {
	SystemFlagNames = append(SystemFlagNames, name)

	return flag.String(name, value, usage)
}
