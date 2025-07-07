package scripting

import (
	"flag"
	"fmt"
	"github.com/ditashi/jsbeautifier-go/jsbeautifier"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/mpetavy/common"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

type ScriptEngine struct {
	Registry *require.Registry
	VM       *goja.Runtime
	program  *goja.Program
}

var (
	fileJsonFormat = flag.Bool("file.json.format", true, "use JSON file format")
)

func NewScriptEngine(src string, modulesPath string) (*ScriptEngine, error) {
	vm := goja.New()

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

	err := registerConsole(registry, vm)
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

func (engine *ScriptEngine) Run(timeout time.Duration, funcName string, args ...any) (goja.Value, error) {
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
			if isTimeout.Load() || err != nil {
				return err
			}

			if funcName != "" {
				fn, ok := goja.AssertFunction(engine.VM.Get(funcName))
				if !ok {
					return fmt.Errorf("undefined function %s", funcName)
				}

				list := []goja.Value{}
				for _, arg := range args {
					list = append(list, engine.VM.ToValue(arg))
				}

				value, err = fn(goja.Undefined(), list...)
				if err != nil {
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

	timeoutCh := time.After(timeout)
	if timeout == 0 {
		timeoutCh = nil
	}

	select {
	case <-timeoutCh:
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
	if !*fileJsonFormat {
		return src, nil
	}

	cpySrc := src

	formatScript, err := jsbeautifier.Beautify(&cpySrc, jsbeautifier.DefaultOptions())
	if common.Error(err) {

	}

	return formatScript, nil
}
