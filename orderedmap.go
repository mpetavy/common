package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type item[K comparable, V any] struct {
	key   K
	value V
}

type OrderedMap[K comparable, V any] struct {
	items []*item[K, V]
}

func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] {
	return &OrderedMap[K, V]{
		items: make([]*item[K, V], 0),
	}
}

func (om *OrderedMap[K, V]) Len() int {
	return len(om.items)
}

func (om *OrderedMap[K, V]) index(key K) int {
	for i := 0; i < len(om.items); i++ {
		if om.items[i].key == key {
			return i
		}
	}

	return -1
}

func (om *OrderedMap[K, V]) GetByIndex(index int) (K, V) {
	return om.items[index].key, om.items[index].value
}

func (om *OrderedMap[K, V]) RemoveByIndex(index int) (K, V) {
	item := om.items[index]

	om.items = SliceDelete(om.items, index)

	return item.key, item.value
}

func (om *OrderedMap[K, V]) Get(key K) (V, bool) {
	index := om.index(key)

	if index == -1 {
		return *(new(V)), false
	}

	return om.items[index].value, true
}

func (om *OrderedMap[K, V]) Add(key K, value V) *OrderedMap[K, V] {
	om.items = append(om.items, &item[K, V]{
		key:   key,
		value: value,
	})

	return om
}

func (om *OrderedMap[K, V]) Remove(key K) V {
	index := om.index(key)
	item := om.items[index]

	om.items = SliceDelete(om.items, index)

	return item.value
}

func (om *OrderedMap[K, V]) Clear() *OrderedMap[K, V] {
	om.items = make([]*item[K, V], 0)

	return om
}

func (om *OrderedMap[K, V]) Keys() []K {
	keys := make([]K, len(om.items))
	for i := 0; i < len(om.items); i++ {
		keys[i] = om.items[i].key
	}

	return keys
}

func (om *OrderedMap[K, V]) Sort(fn func(K, K) bool) {
	sort.SliceStable(om.items, func(i, j int) bool {
		return fn(om.items[i].key, om.items[j].key)
	})
}

func (om *OrderedMap[K, V]) Range(fn func(K, V)) {
	for i := 0; i < len(om.items); i++ {
		fn(om.items[i].key, om.items[i].value)
	}
}

func (om *OrderedMap[K, V]) MarshalJSON() ([]byte, error) {
	buf := bytes.Buffer{}

	buf.WriteString("{")
	for i := 0; i < len(om.items); i++ {
		if i > 0 {
			buf.WriteString(",")
		}

		kj, err := json.Marshal(om.items[i].key)
		if err != nil {
			return nil, err
		}

		vj, err := json.Marshal(om.items[i].value)
		if err != nil {
			return nil, err
		}

		kk := string(kj)
		if !strings.HasPrefix(kk, "\"") {
			kk = "\"" + kk
		}
		if !strings.HasSuffix(kk, "\"") {
			kk = kk + "\""
		}
		vv := string(vj)

		buf.WriteString(fmt.Sprintf("%s:", kk))

		isObj := reflect.TypeOf(om.items[i].value).Kind() == reflect.Slice || reflect.TypeOf(om.items[i].value).Kind() == reflect.Map

		if isObj {
			buf.WriteString("{")
		}

		buf.WriteString(vv)

		if isObj {
			buf.WriteString("}")
		}
	}
	buf.WriteString("}")

	return buf.Bytes(), nil
}

//func (om *OrderedMap[K, V]) Unmarshal(data []byte, v any) error {
//	var m map[string]interface{}
//
//	err := json.Unmarshal(data, &m)
//	if common.Error(err) {
//		return err
//	}
//
//	for k, v := range m {
//
//	}
//
//	return nil
//}
