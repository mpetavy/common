package generics

import (
	"container/list"
	"fmt"
	"reflect"
)

const (
	EQUAL int = iota
	SMALLER
	GREATER
)

type ErrDifferentType struct {
	at, bt reflect.Type
}

func (e *ErrDifferentType) Error() string {
	return fmt.Sprintf("Different type: %s != %s", e.at.Name(), e.at.Name())
}

type ErrIndexOutOfRange struct {
	index, max int
}

func (e *ErrIndexOutOfRange) Error() string {
	return fmt.Sprintf("Index out of range: %d; max: %d", e.index, e.max)
}

type Comparator func(a, b interface{}) (int, error)

func StringComparator() Comparator {
	return func(a, b interface{}) (int, error) {
		var t string
		err := checkType(a, b, reflect.ValueOf(t).Type())
		if err != nil {
			return 0, err
		}

		_a := a.(string)
		_b := a.(string)

		if _a < _b {
			return SMALLER, nil
		} else if _a > _b {
			return GREATER, nil
		} else {
			return EQUAL, nil
		}
	}
}

func IntegerComparator() Comparator {
	return func(a, b interface{}) (int, error) {
		var t int
		err := checkType(a, b, reflect.ValueOf(t).Type())
		if err != nil {
			return 0, err
		}

		_a := a.(int)
		_b := a.(int)

		if _a < _b {
			return SMALLER, nil
		} else if _a > _b {
			return GREATER, nil
		} else {
			return EQUAL, nil
		}
	}
}

func checkType(a, b interface{}, typ reflect.Type) error {
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	if va.Type() != vb.Type() {
		return &ErrDifferentType{va.Type(), va.Type()}
	}

	if va.Type().Kind() != typ.Kind() {
		return &ErrDifferentType{va.Type(), typ}
	}

	return nil
}

func FindInList(list list.List, search interface{}, comparator Comparator) (int, error) {
	i := 0
	for e := list.Front(); e != nil; e = e.Next() {
		c, err := comparator(e.Value, search)
		if err != nil {
			return -1, err
		}
		if c == EQUAL {
			return i, nil
		}
		i++
	}

	return -1, nil
}

func GetFromList(list list.List, index int) interface{} {
	if index < 0 || index >= list.Len() {
		panic(&ErrIndexOutOfRange{index: index, max: list.Len()})
	}

	i := 0
	for e := list.Front(); e != nil; e = e.Next() {
		if i == index {
			return e.Value
		}
		i++
	}

	return nil
}

func FindInSlice(slice []interface{}, search interface{}, comparator Comparator) (int, error) {
	i := 0
	for _, e := range slice {
		c, err := comparator(e, search)
		if err != nil {
			return -1, err
		}
		if c == EQUAL {
			return i, nil
		}
		i++
	}

	return -1, nil
}
