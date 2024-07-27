package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dlclark/regexp2"
	"regexp"
	"strings"
)

//https://jsoneditoronline.org/

type Jason struct {
	attributes map[string]interface{}
}

func NewJason(s string) (*Jason, error) {
	var m map[string]interface{}

	s, err := RemoveJsonComments(s)
	if Error(err) {
		return nil, err
	}

	err = json.Unmarshal([]byte(s), &m)
	if Error(err) {
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

	s, ok := o.([]interface{})
	if ok {
		m, _ := s[0].(map[string]interface{})

		return &Jason{m}, nil
	}

	m, ok := o.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("not an object for key: %s", key)
	}

	return &Jason{m}, nil
}

func (jason *Jason) ElementByPath(path string) (*Jason, error) {
	splits := Split(path, "/")

	var err error
	j := jason
	for _, split := range splits {
		j, err = j.Element(split)
		if err != nil {
			return j, err
		}
	}

	return j, nil
}

func (jason *Jason) IsString(key string) bool {
	v, err := jason.get(key)
	if err != nil {
		return false
	}

	_, b := v.(string)

	return b
}

func (jason *Jason) IsInt(key string) bool {
	v, err := jason.get(key)
	if err != nil {
		return false
	}

	v1, b := v.(float64)

	return b && (v1 == float64(int(v1)))
}

func (jason *Jason) IsBool(key string) bool {
	v, err := jason.get(key)
	if err != nil {
		return false
	}

	_, b := v.(bool)

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
	if Error(err) {
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
	if Error(err) {
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
	if v == nil || err != nil {
		if len(def) > 0 {
			return def[0], nil
		} else {
			return "", err
		}
	}

	return v.(string), nil
}

func (jason *Jason) Int(key string, def ...int) (result int, err error) {
	defer func() {
		if err != nil {
			if len(def) > 0 {
				result = def[0]
				err = nil
			} else {
				result = 0
			}
		}
	}()

	v, err := jason.get(key)

	v1, b := v.(float64)
	if !b {
		err = fmt.Errorf("not a int: %v", v)
	}

	return int(v1), nil
}

func (jason *Jason) Bool(key string, def ...bool) (result bool, err error) {
	defer func() {
		if err != nil {
			if len(def) > 0 {
				result = def[0]
				err = nil
			} else {
				result = false
			}
		}
	}()

	v, err := jason.get(key)

	v1, b := v.(bool)
	if !b {
		err = fmt.Errorf("not a bool: %v", v)
	}

	return v1, nil
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
			if Error(err) {
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
			if Error(err) {
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

func RemoveJsonComments(s string) (string, error) {
	// enable multiline mode
	// skip from start of line to the first \\ and remove the remaining characters

	s = regexp.MustCompile("(?m)(^ *\t*)\\/\\/.*").ReplaceAllString(s, "")

	// remove a pending , on the last element before a closing ) ] or }
	var err error
	s, err = regexp2.MustCompile(",(?=\\s*[\\)\\]\\}])", 0).Replace(s, "", -1, -1)
	if Error(err) {
		return "", err
	}

	r := bytes.Buffer{}

	scanner := bufio.NewScanner(strings.NewReader(s))
	scanner.Split(ScanLinesWithLF)
	for scanner.Scan() {
		line := scanner.Text()

		if len(strings.TrimSpace(line)) != 0 {
			r.Write([]byte(line))
		}
	}

	return r.String(), nil
}
