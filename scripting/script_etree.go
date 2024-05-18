package scripting

import (
	"github.com/beevik/etree"
	"github.com/dop251/goja"
	"github.com/mpetavy/common"
	"reflect"
)

func registerEtree(vm *goja.Runtime) error {
	d := etree.NewDocument()

	obj := vm.NewObject()

	t := reflect.TypeOf(d)
	v := reflect.ValueOf(d)
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)

		err := obj.Set(m.Name, v.MethodByName(m.Name).Interface())
		if common.Error(err) {
			return err
		}
	}

	err := vm.Set("etree", obj)
	if common.Error(err) {
		return err
	}

	return nil
}
