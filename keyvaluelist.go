package common

import (
	"fmt"
	"sort"
	"strings"
)

type KeyValueList []string

func (kvl *KeyValueList) find(key string) int {
	for i, l := range *kvl {
		ss := strings.Split(l, "=")
		if ss[0] == key {
			return i
		}
	}

	return -1
}

func getValue(line string) string {
	return line[strings.Index(line, "=")+1:]
}

func getKey(line string) string {
	return line[:strings.Index(line, "=")]
}

func (kvl *KeyValueList) Put(key string, value string) error {
	if key == "" {
		return fmt.Errorf("key cannot be null")
	}

	item := fmt.Sprintf("%s=%s", key, value)
	index := kvl.find(key)

	if index == -1 {
		*kvl = append(*kvl, item)
	} else {
		(*kvl)[index] = item
	}

	sort.Strings(*kvl)

	return nil
}

func (kvl *KeyValueList) Get(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("key cannot be null")
	}

	index := kvl.find(key)

	if index == -1 {
		return "", fmt.Errorf("key not found")
	}

	return getValue((*kvl)[index]), nil
}

func (kvl *KeyValueList) Remove(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("key cannot be null")
	}

	index := kvl.find(key)

	if index == -1 {
		return "", fmt.Errorf("key not found")
	}

	item := getValue((*kvl)[index])

	*kvl = append((*kvl)[:index], (*kvl)[index+1:]...)

	return item, nil
}

func (kvl *KeyValueList) Keys() []string {
	keys := make([]string,0)

	for _,kv := range *kvl {
		keys = append(keys,getKey(kv))
	}

	return keys
}

func (kvl *KeyValueList) Values() []string {
	keys := make([]string,0)

	for _,kv := range *kvl {
		keys = append(keys,getValue(kv))
	}

	return keys
}