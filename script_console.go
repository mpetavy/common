package common

import (
	"fmt"
	"github.com/dop251/goja"
	"strings"
)

type gojaConsole struct{}

func (c *gojaConsole) printf(format string, args ...any) {
	if len(args) == 0 {
		fmt.Printf(format)
	} else {
		fmt.Printf(format, args...)
	}
}

func (c *gojaConsole) error(msgs ...string) {
	Error(fmt.Errorf("%s", strings.Join(msgs, " ")))
}

func (c *gojaConsole) info(msgs ...string) {
	Info(strings.Join(msgs, " "))
}

func (c *gojaConsole) debug(msgs ...string) {
	Debug(strings.Join(msgs, " "))
}

func (c *gojaConsole) warn(msgs ...string) {
	Warn(strings.Join(msgs, " "))
}

func (c *gojaConsole) log(msgs ...string) {
	Info(strings.Join(msgs, " "))
}

func registerConsole(vm *goja.Runtime) error {
	c := &gojaConsole{}

	obj := vm.NewObject()

	err := obj.Set("printf", c.printf)
	if Error(err) {
		return err
	}

	err = obj.Set("error", c.error)
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
