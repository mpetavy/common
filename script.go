package common

import (
	"fmt"
	"github.com/dop251/goja"
	req "github.com/dop251/goja_nodejs/require"
	"time"
)

type ScriptEngine struct {
	VM      *goja.Runtime
	program *goja.Program
}

type console struct{}

func (c *console) error(msg string) {
	Error(fmt.Errorf("%s", msg))
}

func (c *console) info(msg string) {
	Info(msg)
}

func (c *console) log(msg string) {
	Debug(msg)
}

func registerConsole(vm *goja.Runtime) error {
	c := &console{}

	console := vm.NewObject()
	err := console.Set("error", c.error)
	if Error(err) {
		return err
	}

	err = console.Set("info", c.info)
	if Error(err) {
		return err
	}

	err = console.Set("log", c.log)
	if Error(err) {
		return err
	}

	err = vm.Set("console", console)
	if Error(err) {
		return err
	}

	return nil
}

func NewScriptEngine(src string) (*ScriptEngine, error) {
	vm := goja.New()

	err := registerConsole(vm)
	if Error(err) {
		return nil, err
	}

	new(req.Registry).Enable(vm)

	program, err := goja.Compile("", src, true)
	if Error(err) {
		return nil, err
	}

	engine := &ScriptEngine{
		VM:      vm,
		program: program,
	}

	return engine, nil
}

func (engine *ScriptEngine) Run(timeout time.Duration, funcName string, input string) (string, error) {
	type result struct {
		value string
		err   error
	}

	ch := make(chan result)

	go func() {
		var value goja.Value
		var err error

		value, err = engine.VM.RunProgram(engine.program)

		if funcName != "" {
			fn := func() (goja.Value, error) {
				var jsFunc func(goja.Value) goja.Value

				err := engine.VM.ExportTo(engine.VM.GlobalObject().Get(funcName), &jsFunc)
				if Error(err) {
					return goja.Undefined(), err
				}

				err = Catch(func() {
					value = jsFunc(engine.VM.ToValue(input))
				})
				if Error(err) {
					return goja.Undefined(), err
				}

				return value, nil
			}

			value, err = fn()
		}

		if err != nil {
			ch <- result{
				value: "",
				err:   err,
			}

			return
		}

		ch <- result{
			value: value.String(),
			err:   nil,
		}
	}()

	select {
	case <-time.After(timeout):
		engine.VM.Interrupt(nil)
		return "", &ErrTimeout{
			Duration: timeout,
			Err:      nil,
		}
	case result := <-ch:
		return result.value, result.err
	}
}
