package common

import (
	"reflect"
)

type OrderedMap struct {
	m map[interface{}]interface{}
	l []interface{}
}

type ErrInvalidType struct {
	Msg string
}

func (e *ErrInvalidType) Error() string {
	return e.Msg
}

func NewOrderedMap(m ...interface{}) *OrderedMap {
	o := OrderedMap{make(map[interface{}]interface{}), make([]interface{}, 0)}

	if len(m) > 0 {
		o.SetMap(m[0])
	}

	return &o
}

func (o *OrderedMap) Clear() *OrderedMap {
	o.m = make(map[interface{}]interface{})
	o.l = make([]interface{}, 0)

	return o
}

func (o *OrderedMap) SetMap(m ...interface{}) *OrderedMap {
	o.Clear()

	if len(m) > 1 || reflect.TypeOf(m[0]).Kind() != reflect.Map {
		panic(&ErrInvalidType{"not a valid map type provided"})

	}

	if len(m) == 1 {
		v := reflect.ValueOf(m[0])

		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)
			o.Set(key.Interface(), value.Interface())
		}
	}

	return o
}

func (o *OrderedMap) Set(key interface{}, value interface{}) *OrderedMap {
	delete(o.m, key)

	o.m[key] = value
	o.l = append(o.l, key)

	return o
}

func (o *OrderedMap) Get(key interface{}) (interface{}, bool) {
	value, ok := o.m[key]

	return value, ok
}

func (o *OrderedMap) Delete(key interface{}) *OrderedMap {
	delete(o.m, key)

	for i, k := range o.l {
		if key == k {
			o.l = append(o.l[:i], o.l[i+1:]...)
			break
		}
	}

	return o
}

func (o *OrderedMap) Len() int {
	return len(o.m)
}

func (o *OrderedMap) Keys() []interface{} {
	return o.l
}
