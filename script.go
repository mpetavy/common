package common

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type ScriptEngine struct {
	Registry *require.Registry
	VM       *goja.Runtime
	program  *goja.Program
}

func (c *console) table(data interface{}) {
	val, ok := data.(reflect.Value)
	if !ok {
		val = reflect.Indirect(reflect.ValueOf(data))
	}

	st := NewStringTable()
	st.AddCols("field", "value")

	switch val.Type().Kind() {
	case reflect.Map:
		iter := val.MapRange()
		for iter.Next() {
			k := iter.Key()
			v := iter.Value()

			st.AddCols(k, fmt.Sprintf("%+v", v.Elem()))
		}
	case reflect.Struct:
		err := IterateStruct(data, func(fieldPath string, fieldType reflect.StructField, fieldValue reflect.Value) error {
			st.AddCols(fieldPath, fmt.Sprintf("%+v", fieldValue.Elem()))

			return nil
		})
		if Error(err) {
			return
		}
	case reflect.Array:
		for i := 0; i < val.Len(); i++ {
			st.AddCols(strconv.Itoa(i), val.Index(i))
		}
	case reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			st.AddCols(strconv.Itoa(i), val.Slice(i, i+1))
		}
	default:
		Error(TraceError(fmt.Errorf("unsupported type")))
	}

	Debug(st.String())
}

func NewScriptEngine(src string, modulesPath string) (*ScriptEngine, error) {
	vm := goja.New()

	err := registerConsole(vm)
	if Error(err) {
		return nil, err
	}

	options := []require.Option{}

	if modulesPath != "" {
		options = append(options, require.WithGlobalFolders(modulesPath))
	}

	options = append(options, require.WithLoader(func(path string) ([]byte, error) {
		resPath := path
		p := strings.Index(resPath, "node_modules")
		if p != -1 {
			resPath = resPath[p:]
		}
		resPath = fmt.Sprintf("js/%s", resPath)
		ba, _, _ := ReadResource(resPath)

		if ba != nil {
			Debug("load module as resource : %s", resPath)

			return ba, nil
		}

		return require.DefaultSourceLoader(path)
	}),
	)

	registry := require.NewRegistry(options...)
	registry.Enable(vm)

	program, err := goja.Compile("", src, false)
	if Error(err) {
		return nil, err
	}

	engine := &ScriptEngine{
		Registry: registry,
		VM:       vm,
		program:  program,
	}

	return engine, nil
}

func (engine *ScriptEngine) Run(timeout time.Duration, funcName string, args any) (goja.Value, error) {
	type result struct {
		value goja.Value
		err   error
	}

	ch := make(chan result)

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine(1))

		var value goja.Value

		err := Catch(func() error {
			var err error

			engine.VM.ClearInterrupt()

			// script must be run once to initialize all functions (also the "main" function)
			value, err = engine.VM.RunProgram(engine.program)
			if Error(err) {
				return err
			}

			if funcName != "" {
				fn, ok := goja.AssertFunction(engine.VM.Get(funcName))
				if !ok {
					return fmt.Errorf("undefined function %s", funcName)
				}

				value, err = fn(goja.Undefined(), engine.VM.ToValue(args))
				if Error(err) {
					return err
				}
			}

			return nil
		})

		if err != nil {
			ch <- result{
				value: nil,
				err:   err,
			}

			return
		}

		ch <- result{
			value: value,
			err:   nil,
		}
	}()

	select {
	case <-time.After(timeout):
		engine.VM.Interrupt(nil)
		return nil, &ErrTimeout{
			Duration: timeout,
			Err:      nil,
		}
	case result := <-ch:
		return result.value, result.err
	}
}

func FormatJavascriptCode(src string) (string, error) {
	beautifyCode, err := embedfs.ReadFile("embed/js/beautify.js")
	if WarnError(err) {
		return src, nil
	}

	se, err := NewScriptEngine(string(beautifyCode), "")
	if Error(err) {
		return "", err
	}

	v, err := se.Run(time.Second, "js_beautify", src)
	if Error(err) {
		return "", err
	}

	code := v.String()
	code = strings.Replace(code, "= >", "=>", -1)
	code = strings.Replace(code, "\r\n", "\n", -1)

	return code, nil
}
