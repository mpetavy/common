package common

import (
	"github.com/dop251/goja"
	req "github.com/dop251/goja_nodejs/require"
	"github.com/robertkrimen/otto"
	"strings"
	"time"
)

type ScriptEngine interface {
	Run(time.Duration) (string, error)
}

type ottoEngine struct {
	ScriptEngine

	engine *otto.Otto
	code   *otto.Script
}

func NewOttoEngine(src string) (ScriptEngine, error) {
	vm := otto.New()
	vm.Set("__log__", func(call otto.FunctionCall) otto.Value {
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

func (engine *ottoEngine) Run(timeout time.Duration) (result string, err error) {
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

	result, err = value.ToString()
	if Error(err) {
		return "", err
	}

	return result, nil
}

type gojaEngine struct {
	ScriptEngine

	engine *goja.Runtime
	code   *goja.Program
}

type console struct{}

func (c *console) log(msg string) {
	Debug(msg)
}

func newConsole(vm *goja.Runtime) *goja.Object {
	c := &console{}

	obj := vm.NewObject()
	obj.Set("log", c.log)

	return obj
}

func NewGojaEngine(src string) (ScriptEngine, error) {
	vm := goja.New()

	vm.Set("console", newConsole(vm))

	new(req.Registry).Enable(vm)

	prog, err := goja.Compile("", src, true)
	if Error(err) {
		return nil, err
	}

	engine := &gojaEngine{
		engine: vm,
		code:   prog,
	}

	return engine, nil
}

func (engine *gojaEngine) Run(timeout time.Duration) (string, error) {
	type result struct {
		value string
		err   error
	}

	ch := make(chan result)

	go func() {
		value, err := engine.engine.RunProgram(engine.code)
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
		engine.engine.Interrupt(nil)
		return "", &ErrTimeout{
			Duration: timeout,
			Err:      nil,
		}
	case result := <-ch:
		return result.value, result.err
	}
}
