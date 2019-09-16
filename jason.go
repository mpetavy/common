package common

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

//https://jsoneditoronline.org/

type Jason struct {
	attributes map[string]interface{}
}

func NewJason(s string) (*Jason, error) {
	var m map[string]interface{}

	err := json.Unmarshal([]byte(RemoveJsonComments(s)), &m)
	if err != nil {
		return nil, err
	}

	return &Jason{m}, nil
}

func (jason *Jason) Count() int {
	return len(jason.attributes)
}

func (jason *Jason) Exists(key string) bool {
	_, ok := jason.attributes[key]

	return ok
}

func (jason *Jason) Elements() []string {
	var result []string

	for k := range jason.attributes {
		result = append(result, k)
	}

	return result
}

func (jason *Jason) Element(key string) (*Jason, error) {
	o, ok := jason.attributes[key]

	if !ok {
		return nil, fmt.Errorf("object not found for key: %s", key)
	}

	m, ok := o.(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf("not an object for key: %s", key)
	}

	return &Jason{m}, nil
}

func (jason *Jason) IsString(key string) bool {
	return !jason.IsInt(key) && !jason.IsBool(key)
}

func (jason *Jason) IsInt(key string) bool {
	v, err := jason.get(key)
	if err != nil {
		return false
	}

	_, b := v.(int)

	if !b {
		_, err = strconv.Atoi(v.(string))

		b = err == nil
	}

	return b
}

func (jason *Jason) IsBool(key string) bool {
	v, err := jason.get(key)
	if err != nil {
		return false
	}

	_, b := v.(bool)

	if !b {
		b = ToBool(v.(string))
	}

	return b
}

func (jason *Jason) get(key string) (interface{}, error) {
	value, ok := jason.attributes[key]
	if !ok {
		return "", fmt.Errorf("no value found for key: %s", key)
	}

	return value, nil
}

func (jason *Jason) IsArray(key string) bool {
	v, err := jason.get(key)
	if err != nil {
		return false
	}

	_, ok := v.([]interface{})

	return ok
}

func (jason *Jason) ArrayCount(key string) int {
	v, err := jason.get(key)
	if err != nil {
		return 0
	}

	a, ok := v.([]interface{})

	if !ok {
		return 0
	}

	return len(a)
}

func (jason *Jason) Array(key string, index int) (*Jason, error) {
	v, err := jason.get(key)
	if err != nil {
		return nil, err
	}

	a, ok := v.([]interface{})

	m, ok := a[index].(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf("not an object for key: %s", key)
	}

	return &Jason{m}, nil
}

func (jason *Jason) String(key string, def ...string) (string, error) {
	v, err := jason.get(key)
	if err != nil {
		if len(def) > 0 {
			return def[0], nil
		} else {
			return "", err
		}
	}

	return v.(string), nil
}

func (jason *Jason) Int(key string, def ...int) (int, error) {
	v, err := jason.get(key)
	if err != nil {
		if len(def) > 0 {
			return def[0], nil
		} else {
			return 0, err
		}
	}

	i, err := strconv.Atoi(v.(string))
	if err != nil {
		return 0, err
	}

	return i, nil
}

func (jason *Jason) Bool(key string, def ...bool) (bool, error) {
	v, err := jason.get(key)
	if err != nil {
		if len(def) > 0 {
			return def[0], nil
		} else {
			return false, err
		}
	}

	return ToBool(v.(string)), nil
}

func (jason *Jason) pretty(index int) (string, error) {
	tab := strings.Repeat(" ", index*4)
	s := ""

	for k, v := range jason.attributes {
		if v == nil {
			s += fmt.Sprintf("%s\"%s\": null\n", tab, k)

			continue
		}

		m, ok := v.(map[string]interface{})
		if ok {
			jason := &Jason{m}
			ss, err := jason.pretty(index + 1)
			if err != nil {
				return "", err
			}

			s += fmt.Sprintf("%s\"%s\": {\n", tab, k)
			s += ss
			s += fmt.Sprintf("%s}\n", tab)

			continue
		}

		a, ok := v.([]interface{})
		if ok {
			ss, err := ToStrings(a)
			if err != nil {
				return "", err
			}
			s += fmt.Sprintf("%s\"%s\": [%v]\n", tab, k, strings.Join(SurroundWith(ss, "\""), ","))
		} else {
			s += fmt.Sprintf("%s\"%s\": \"%v\"\n", tab, k, v)
		}
	}

	return s, nil
}

func (jason *Jason) Pretty() (string, error) {
	return jason.pretty(0)
}

func RemoveJsonComments(s string) string {
	s = regexp.MustCompile("(?s)\\/\\*.*?\\*\\/").ReplaceAllString(s, "")
	s = regexp.MustCompile("[^:]\\/\\/.*").ReplaceAllString(s, "")

	return s
}
