package scripting

import (
	"embed"
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/mpetavy/common"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

//go:embed embed/*
var embedfs embed.FS

type ScriptEngine struct {
	Registry *require.Registry
	VM       *goja.Runtime
	program  *goja.Program
}

func (c *gojaConsole) table(data interface{}) {
	val, ok := data.(reflect.Value)
	if !ok {
		val = reflect.Indirect(reflect.ValueOf(data))
	}

	st := common.NewStringTable()
	st.AddCols("field", "value")

	switch val.Type().Kind() {
	case reflect.Map:
		iter := val.MapRange()
		for iter.Next() {
			k := iter.Key()
			v := iter.Value()

			st.AddCols(k.String(), fmt.Sprintf("%+v", v.Elem()))
		}
	case reflect.Struct:
		err := common.IterateStruct(data, func(fieldPath string, fieldType reflect.StructField, fieldValue reflect.Value) error {
			st.AddCols(fieldPath, fmt.Sprintf("%+v", fieldValue.Elem()))

			return nil
		})
		if common.Error(err) {
			return
		}
	case reflect.Array:
		for i := 0; i < val.Len(); i++ {
			st.AddCols(strconv.Itoa(i), val.Index(i).String())
		}
	case reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			st.AddCols(strconv.Itoa(i), val.Slice(i, i+1).String())
		}
	default:
		common.Error(common.TraceError(fmt.Errorf("unsupported type")))
	}

	common.Debug(st.Table())
}

func NewScriptEngine(src string, modulesPath string) (*ScriptEngine, error) {
	vm := goja.New()

	err := registerConsole(vm)
	if common.Error(err) {
		return nil, err
	}

	err = registerHttp(vm)
	if common.Error(err) {
		return nil, err
	}

	err = registerEtree(vm)
	if common.Error(err) {
		return nil, err
	}

	options := []require.Option{}

	if modulesPath != "" {
		options = append(options, require.WithGlobalFolders(modulesPath))
	}

	options = append(options, require.WithLoader(func(path string) ([]byte, error) {
		ba, err := require.DefaultSourceLoader(path)
		if err == nil {
			common.Debug("load Javascript module as file: %s", path)

			return ba, err
		}

		resPath := path
		p := strings.Index(resPath, "node_modules")
		if p != -1 {
			resPath = resPath[p:]
		}

		resPath = strings.ReplaceAll(filepath.Join("node", resPath), "\\", "/")

		ba, _, err = common.ReadResource(resPath)
		if ba != nil {
			common.Debug("load Javascript module as embedded resource: %s -> %s", path, resPath)

			return ba, nil
		}

		return nil, require.ModuleFileDoesNotExistError
	}),
	)

	registry := require.NewRegistry(options...)
	registry.Enable(vm)

	program, err := goja.Compile("", src, false)
	if common.Error(err) {
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

	var isTimeout atomic.Bool

	ch := make(chan result)

	go func() {
		defer common.UnregisterGoRoutine(common.RegisterGoRoutine(1))

		var value goja.Value

		err := common.Catch(func() error {
			var err error

			engine.VM.ClearInterrupt()

			// script must be run once to initialize all functions (also the "main" function)
			value, err = engine.VM.RunProgram(engine.program)
			if isTimeout.Load() || common.DebugError(err) {
				return err
			}

			if funcName != "" {
				fn, ok := goja.AssertFunction(engine.VM.Get(funcName))
				if !ok {
					return fmt.Errorf("undefined function %s", funcName)
				}

				value, err = fn(goja.Undefined(), engine.VM.ToValue(args))
				if common.DebugError(err) {
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
		isTimeout.Store(true)

		engine.VM.Interrupt(nil)

		return nil, &common.ErrTimeout{
			Duration: timeout,
			Err:      nil,
		}
	case result := <-ch:
		return result.value, result.err
	}
}

func FormatJavascriptCode(src string) (string, error) {
	beautifyCode, err := embedfs.ReadFile("embed/js/beautify.js")
	if common.WarnError(err) {
		return src, nil
	}

	se, err := NewScriptEngine(string(beautifyCode), "")
	if common.Error(err) {
		return "", err
	}

	v, err := se.Run(time.Second, "js_beautify", src)
	if common.Error(err) {
		return "", err
	}

	code := v.String()
	code = strings.Replace(code, "= >", "=>", -1)
	code = strings.Replace(code, "\r\n", "\n", -1)

	return code, nil
}
