package common

import (
	"fmt"
	"github.com/dop251/goja"
	"strings"
)

type gojaConsole struct{}

func format(args ...any) string {
	list := []string{}

	for _, arg := range args {
		list = append(list, fmt.Sprintf("%+v", arg))
	}

	return strings.Join(list, " ")
}

func (c *gojaConsole) error(args ...any) {
	Error(fmt.Errorf("%s", format(args...)))
}

func (c *gojaConsole) info(args ...any) {
	Info(format(args...))
}

func (c *gojaConsole) debug(args ...any) {
	Debug(format(args...))
}

func (c *gojaConsole) warn(args ...any) {
	Warn(format(args...))
}

func (c *gojaConsole) log(args ...any) {
	Debug(format(args...))
}

func registerConsole(vm *goja.Runtime) error {
	c := &gojaConsole{}

	obj := vm.NewObject()

	err := obj.Set("error", c.error)
	if Error(err) {
		return err
	}

	err = obj.Set("debug", c.debug)
	if Error(err) {
		return err
	}

	err = obj.Set("warn", c.warn)
	if Error(err) {
		return err
	}

	err = obj.Set("info", c.info)
	if Error(err) {
		return err
	}

	err = obj.Set("log", c.log)
	if Error(err) {
		return err
	}

	err = obj.Set("table", c.table)
	if Error(err) {
		return err
	}

	err = vm.Set("console", obj)
	if Error(err) {
		return err
	}

	return nil
}
