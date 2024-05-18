package scripting

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/mpetavy/common"
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
	common.Error(fmt.Errorf("%s", format(args...)))
}

func (c *gojaConsole) info(args ...any) {
	common.Info(format(args...))
}

func (c *gojaConsole) debug(args ...any) {
	common.Debug(format(args...))
}

func (c *gojaConsole) warn(args ...any) {
	common.Warn(format(args...))
}

func (c *gojaConsole) log(args ...any) {
	common.Debug(format(args...))
}

func registerConsole(vm *goja.Runtime) error {
	c := &gojaConsole{}

	obj := vm.NewObject()

	err := obj.Set("error", c.error)
	if common.Error(err) {
		return err
	}

	err = obj.Set("debug", c.debug)
	if common.Error(err) {
		return err
	}

	err = obj.Set("warn", c.warn)
	if common.Error(err) {
		return err
	}

	err = obj.Set("info", c.info)
	if common.Error(err) {
		return err
	}

	err = obj.Set("log", c.log)
	if common.Error(err) {
		return err
	}

	err = obj.Set("table", c.table)
	if common.Error(err) {
		return err
	}

	err = vm.Set("console", obj)
	if common.Error(err) {
		return err
	}

	return nil
}
