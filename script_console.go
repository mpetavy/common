package common

import (
	"fmt"
	"github.com/dop251/goja"
	"strings"
)

type console struct{}

func (c *console) error(msgs ...string) {
	Error(fmt.Errorf("%s", strings.Join(msgs, " ")))
}

func (c *console) info(msgs ...string) {
	Info(strings.Join(msgs, " "))
}

func (c *console) debug(msgs ...string) {
	Debug(strings.Join(msgs, " "))
}

func (c *console) warn(msgs ...string) {
	Warn(strings.Join(msgs, " "))
}

func (c *console) log(msgs ...string) {
	Debug(strings.Join(msgs, " "))
}

func registerConsole(vm *goja.Runtime) error {
	c := &console{}

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
