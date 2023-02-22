package common

import (
	"golang.org/x/exp/slices"
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

	om.items = slices.Delete(om.items, index, index+1)

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

	om.items = slices.Delete(om.items, index, index+1)

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

func (om *OrderedMap[K, V]) Range(fn func(K, V)) {
	for i := 0; i < len(om.items); i++ {
		fn(om.items[i].key, om.items[i].value)
	}
}
