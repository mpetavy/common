package common

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"path/filepath"
	"reflect"
	"strconv"
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

	modulesPath = CleanPath(modulesPath)

	registry := require.NewRegistry(
		require.WithGlobalFolders(modulesPath),
		require.WithLoader(func(path string) ([]byte, error) {
			path = CleanPath(path)

			Debug("load module: %s", path)

			ba, _, _ := ReadResource(filepath.Base(path))

			if ba != nil {
				return ba, nil
			}

			return require.DefaultSourceLoader(path)
		}),
	)

	registry.Enable(vm)

	program, err := goja.Compile("", src, true)
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

func (engine *ScriptEngine) Run(timeout time.Duration, funcName string, input string) (string, error) {
	type result struct {
		value string
		err   error
	}

	ch := make(chan result)

	go func() {
		var value goja.Value

		err := Catch(func() error {
			var err error

			if funcName != "" {
				fn, ok := goja.AssertFunction(engine.VM.Get(funcName))
				if !ok {
					return fmt.Errorf("undefined function %s", funcName)
				}

				value, err = fn(goja.Undefined(), engine.VM.ToValue(input))
				if Error(err) {
					return err
				}
			} else {
				value, err = engine.VM.RunProgram(engine.program)
				if Error(err) {
					return err
				}
			}

			return nil
		})

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
