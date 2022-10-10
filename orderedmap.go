package common

import "golang.org/x/exp/slices"

type OrderedMap[K comparable, V any] struct {
	m map[K]V
	l []K
}

func NewOrderedMap[K comparable, V any]() *OrderedMap[K, V] {
	return &OrderedMap[K, V]{
		m: make(map[K]V),
		l: nil,
	}
}

func (om *OrderedMap[K, V]) Len() int {
	return len(om.l)
}

func (om *OrderedMap[K, V]) Get(k K) V {
	return om.m[k]
}

func (om *OrderedMap[K, V]) GetOk(k K) (V, bool) {
	v, ok := om.m[k]

	return v, ok
}

func (om *OrderedMap[K, V]) Add(k K, v V) *OrderedMap[K, V] {
	om.m[k] = v
	om.l = append(om.l, k)

	return om
}

func (om *OrderedMap[K, V]) Remove(k K) *OrderedMap[K, V] {
	delete(om.m, k)

	p := slices.Index(om.l, k)
	if p != -1 {
		om.l = slices.Delete(om.l, p, p+1)
	}

	return om
}

func (om *OrderedMap[K, V]) Clear() *OrderedMap[K, V] {
	om.m = make(map[K]V)

	return om
}

func (om *OrderedMap[K, V]) Keys() []K {
	lcopy := make([]K, len(om.l))
	copy(lcopy, om.l)

	return om.l
}

func (om *OrderedMap[K, V]) Range(fn func(K, V)) {
	for _, k := range om.l {
		fn(k, om.m[k])
	}
}
