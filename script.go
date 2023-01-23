package common

import (
	"github.com/robertkrimen/otto"
	"strings"
	"time"
)

type ScriptEngine struct {
	otto   *otto.Otto
	script *otto.Script
}

var timeoutError = &ErrTimeout{}

func NewScriptEngine(src string) (*ScriptEngine, error) {
	o := otto.New()
	o.Set("__log__", func(call otto.FunctionCall) otto.Value {
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

	s, err := o.Compile("", "console.log = __log__;"+src)
	if Error(err) {
		return nil, err
	}

	engine := &ScriptEngine{
		otto:   o,
		script: s,
	}
	return engine, nil
}

func (engine *ScriptEngine) Run(timeout time.Duration) (value otto.Value, err error) {
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

	engine.otto.Interrupt = make(chan func(), 1)
	watchdogCleanup := make(chan struct{})
	defer close(watchdogCleanup)

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine(1))

		select {
		case <-time.After(timeout):
			engine.otto.Interrupt <- func() {
				panic(timeoutErr)
			}
		case <-watchdogCleanup:
		}
		close(engine.otto.Interrupt)
	}()

	value, err = engine.otto.Run(engine.script) // Here be dragons (risky code)
	if Error(err) {
		return value, err
	}

	return value, err
}
