package common

import (
	"fmt"
	"github.com/beevik/etree"
	"github.com/dop251/goja"
	"reflect"
)

func registerEtree(vm *goja.Runtime) error {
	d := etree.NewDocument()

	obj := vm.NewObject()

	t := reflect.TypeOf(d)
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		err := obj.Set(m.Name, m.Func)
		if Error(err) {
			return err
		}
		fmt.Println(m.Name)
	}

	err := vm.Set("etree", obj)
	if Error(err) {
		return err
	}

	return nil
}
