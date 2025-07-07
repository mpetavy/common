package scripting

import (
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/mpetavy/common"
	"reflect"
	"strconv"
	"strings"
)

type logPrinter struct {
	vm *goja.Runtime
}

func (lp logPrinter) Log(t string) {
	common.Debug(t)
}

func (lp logPrinter) Warn(t string) {
	common.Warn(t)
}

func (lp logPrinter) Error(t string) {
	errorstack, _ := lp.vm.RunString("(new Error()).stack")
	stack := errorstack.String()

	lines := common.Split(stack, "\n")
	if len(lines) > 2 {
		stack = strings.Join(lines[3:], "\n")
	}

	common.Error(fmt.Errorf("%s\n%s\n", t, stack))
}

func table(data interface{}) {
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
			if val.Index(i).Type().Kind() == reflect.Array {
				st.AddCols(fmt.Sprintf("%v", val.Index(i).Index(0).Elem()), fmt.Sprintf("%v", val.Index(i).Index(1).Elem()))
			} else {
				st.AddCols(strconv.Itoa(i), fmt.Sprintf("%v", val.Index(i).Elem()))
			}
		}
	default:
		common.Error(common.TraceError(fmt.Errorf("unsupported type")))
	}

	common.Debug(st.Table())
}

func registerConsole(registry *require.Registry, vm *goja.Runtime) error {
	registry.RegisterNativeModule("console", console.RequireWithPrinter(&logPrinter{vm: vm}))

	obj := require.Require(vm, "console")
	err := vm.Set("console", obj)
	if common.Error(err) {
		return err
	}

	err = obj.ToObject(vm).Set("table", func(call goja.FunctionCall) goja.Value {
		obj := call.Argument(0).Export()
		table(obj)

		return goja.Undefined()
	})
	if common.Error(err) {
		return err
	}

	err = obj.ToObject(vm).Set("info", func(call goja.FunctionCall) goja.Value {
		obj := call.Argument(0)
		//table(obj)

		common.Info(obj.String())

		return goja.Undefined()
	})
	if common.Error(err) {
		return err
	}

	return nil
}
