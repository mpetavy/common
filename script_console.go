package common

import (
	"fmt"
	"github.com/dop251/goja"
	"strings"
)

type gojaConsole struct{}

func (c *gojaConsole) error(args ...string) {
	Error(fmt.Errorf("%s", strings.Join(args, " ")))
}

func (c *gojaConsole) info(args ...string) {
	Info(strings.Join(args, " "))
}

func (c *gojaConsole) debug(args ...string) {
	Debug(strings.Join(args, " "))
}

func (c *gojaConsole) warn(args ...string) {
	Warn(strings.Join(args, " "))
}

func (c *gojaConsole) log(args ...string) {
	Debug(strings.Join(args, " "))
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
