package common

import (
	"fmt"
	"github.com/dop251/goja"
	req "github.com/dop251/goja_nodejs/require"
	"github.com/robertkrimen/otto"
	"strings"
	"time"
)

type ScriptEngine interface {
	Run(time.Duration, string, string) (string, error)
}

type ottoEngine struct {
	ScriptEngine

	engine *otto.Otto
	code   *otto.Script
}

func NewOttoEngine(src string) (ScriptEngine, error) {
	vm := otto.New()
	err := vm.Set("__log__", func(call otto.FunctionCall) otto.Value {
		sb := strings.Builder{}
		for _, v := range call.ArgumentList {
			if sb.Len() > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(v.String())

		}

		Debug(sb.String())

		return otto.Value{}
	})
	if Error(err) {
		return nil, err
	}

	prog, err := vm.Compile("", "console.log = __log__;"+src)
	if Error(err) {
		return nil, err
	}

	engine := &ottoEngine{
		engine: vm,
		code:   prog,
	}

	return engine, nil
}

func (engine *ottoEngine) Run(timeout time.Duration, funcName string, input string) (result string, err error) {
	timeoutErr := &ErrTimeout{
		Duration: timeout,
	}

	defer func() {
		if caught := recover(); caught != nil {
			if caught == timeoutErr {
				err = timeoutErr
				return
			}
		}
	}()

	engine.engine.Interrupt = make(chan func(), 1)
	watchdogCleanup := make(chan struct{})
	defer close(watchdogCleanup)

	go func() {
		select {
		case <-time.After(timeout):
			engine.engine.Interrupt <- func() {
				panic(timeoutErr)
			}
		case <-watchdogCleanup:
		}
		close(engine.engine.Interrupt)
	}()

	value, err := engine.engine.Run(engine.code)
	if Error(err) {
		return "", err
	}

	if funcName != "" {
		value, err = engine.engine.Call(funcName, nil, input)
		if Error(err) {
			return "", err
		}
	}

	result, err = value.ToString()
	if Error(err) {
		return "", err
	}

	return result, nil
}

type GojaEngine struct {
	ScriptEngine

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

func NewGojaEngine(src string) (*GojaEngine, error) {
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

	engine := &GojaEngine{
		VM:      vm,
		program: program,
	}

	return engine, nil
}

func (engine *GojaEngine) Run(timeout time.Duration, funcName string, input string) (string, error) {
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
