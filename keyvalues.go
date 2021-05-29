package common

import (
	"fmt"
	"sort"
	"strings"
)

type KeyValue string

func (kv KeyValue) Key() string {
	p := strings.Index(string(kv), "=")

	if p != -1 {
		return string(kv)[:p]
	}

	return string(kv)
}

func (kv KeyValue) Value() string {
	p := strings.Index(string(kv), "=")

	if p != -1 {
		return string(kv)[p+1:]
	}

	return ""
}

type KeyValues []KeyValue

func NewKeyValues(list []string) *KeyValues {
	kvs := &KeyValues{}

	for _, item := range list {
		kvs.Add(KeyValue(item).Key(), KeyValue(item).Value())
	}

	return kvs
}

func (kvs *KeyValues) Index(key string) int {
	for i, item := range *kvs {
		if item.Key() == key {
			return i
		}
	}

	return -1
}

func (kvs *KeyValues) getValue(line string) string {
	return line[strings.Index(line, "=")+1:]
}

func (kvs *KeyValues) getKey(line string) string {
	return line[:strings.Index(line, "=")]
}

func (kvs *KeyValues) Add(key string, value string) error {
	if key == "" {
		return fmt.Errorf("key cannot be null")
	}

	*kvs = append(*kvs, KeyValue(key+"="+value))

	return nil
}

func (kvs *KeyValues) Put(key string, value string) error {
	if key == "" {
		return fmt.Errorf("key cannot be null")
	}

	item := KeyValue(key + "=" + value)
	index := kvs.Index(key)

	if index == -1 {
		*kvs = append(*kvs, item)
	} else {
		(*kvs)[index] = item
	}

	return nil
}

func (kvs *KeyValues) Sort() {
	sort.SliceStable(*kvs, func(i, j int) bool {
		return strings.ToUpper((*kvs)[i].Key()) < strings.ToUpper((*kvs)[i].Key())
	})
}

func (kvs *KeyValues) Contains(key string) bool {
	_, err := kvs.Get(key)

	return err == nil
}

func (kvs *KeyValues) Get(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("key cannot be null")
	}

	index := kvs.Index(key)

	if index == -1 {
		return "", fmt.Errorf("key not found")
	}

	return (*kvs)[index].Value(), nil
}

func (kvs *KeyValues) Remove(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("key cannot be null")
	}

	index := kvs.Index(key)

	if index == -1 {
		return "", fmt.Errorf("key not found")
	}

	item := (*kvs)[index].Value()

	*kvs = append((*kvs)[:index], (*kvs)[index+1:]...)

	return item, nil
}

func (kvs *KeyValues) Keys() []string {
	keys := make([]string, 0)

	for _, kv := range *kvs {
		keys = append(keys, kv.Key())
	}

	return keys
}

func (kvs *KeyValues) Values() []string {
	keys := make([]string, 0)

	for _, kv := range *kvs {
		keys = append(keys, kv.Value())
	}

	return keys
}

func (kvs *KeyValues) Clear() {
	*kvs = nil
}
