package common

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"os"
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

			ba, _, err := ReadResource(filepath.Base(path))

			if ba == nil && modulesPath != "" {
				if filepath.Dir(path) == modulesPath {
					ba, err = os.ReadFile(path)
					if Error(err) {
						return nil, err
					}
				} else {
					return nil, TraceError(require.IllegalModuleNameError)
				}
			}

			if ba == nil {
				return nil, TraceError(require.ModuleFileDoesNotExistError)
			}

			str := fmt.Sprintf(`(function(__filename,__dirname){%s})(%q,%q)`,
				ba,
				path,
				filepath.Dir(path),
			)

			return []byte(str), nil
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
