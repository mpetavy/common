package common

import (
	"github.com/beevik/etree"
	"github.com/dop251/goja"
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
		if Error(err) {
			return err
		}
	}

	err := vm.Set("etree", obj)
	if Error(err) {
		return err
	}

	return nil
}
