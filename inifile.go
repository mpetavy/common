package common

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
)

const (
	DEFAULT_SECTION = "default"
)

type IniFile struct {
	keyValues map[string]map[string]string
}

func NewIniFile() *IniFile {
	return &IniFile{
		keyValues: make(map[string]map[string]string),
	}
}

func (ini *IniFile) LoadFile(filename string) error {
	ba, err := os.ReadFile(filename)
	if Error(err) {
		return err
	}

	err = ini.Load(ba)
	if Error(err) {
		return err
	}

	return nil
}

func (ini *IniFile) Load(ba []byte) error {
	if !bytes.HasSuffix(ba, []byte("\n")) {
		ba = append(ba, '\n')
	}

	withCrlf, err := NewSeparatorSplitFunc(nil, []byte("\n"), false)
	if Error(err) {
		return err
	}

	sectionName := DEFAULT_SECTION

	scanner := bufio.NewScanner(bytes.NewReader(ba))
	scanner.Split(withCrlf)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if len(line) == 0 || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") {
			sectionName = strings.Trim(line, "[]")

			section, ok := ini.keyValues[sectionName]
			if ok && len(section) > 0 {
				return fmt.Errorf("duplicate section: %s", sectionName)
			}

			ini.keyValues[sectionName] = make(map[string]string)

			continue
		}

		p := strings.Index(line, "=")
		if p == -1 {
			continue
		}

		key := strings.TrimSpace(line[0:p])
		value := strings.TrimSpace(line[p+1:])

		var delim string
		switch {
		case strings.HasPrefix(value, "\""):
			delim = "\""
		case strings.HasPrefix(value, "`"):
			delim = "``"
		}

		if delim != "" {
			sb := strings.Builder{}
			sb.WriteString(value)

			for scanner.Scan() {
				line = scanner.Text()
				sb.WriteString(line)

				if strings.HasSuffix(line, fmt.Sprintf("%s\n", delim)) {
					break
				}
			}

			value = strings.TrimSpace(sb.String())
			value = strings.Trim(value, delim)
		}

		if strings.HasPrefix(value, "@") {
			ba, err := os.ReadFile(value[1:])
			if Error(err) {
				return err
			}

			value = string(ba)
		}

		if ini.Get(key, value, sectionName) != "" {
			return fmt.Errorf("duplicate key %s in section %s", key, sectionName)
		}

		ini.Set(key, value, sectionName)
	}

	return nil
}

func (ini *IniFile) Sections() []string {
	sections := []string{}

	for k := range ini.keyValues {
		sections = append(sections, k)
	}

	sort.Strings(sections)

	p := slices.Index(sections, DEFAULT_SECTION)
	if p != -1 {
		sections = slices.Delete(sections, p, p+1)
		sections = slices.Insert(sections, 0, DEFAULT_SECTION)
	}

	return sections
}

func (ini *IniFile) Keys(sectionNames ...string) []string {
	keys := []string{}

	if len(sectionNames) == 0 {
		sectionNames = []string{DEFAULT_SECTION}
	}

	for _, sectionName := range sectionNames {
		for key := range ini.keyValues[sectionName] {
			if !slices.Contains(keys, key) {
				keys = append(keys, key)
			}
		}
	}

	sort.Strings(keys)

	return keys
}

func (ini *IniFile) Get(key string, sectionNames ...string) string {
	if len(sectionNames) == 0 {
		sectionNames = []string{DEFAULT_SECTION}
	}

	value := ""

	for _, sectionName := range sectionNames {
		v, ok := ini.keyValues[sectionName][key]
		if ok {
			value = v
		}
	}

	return value
}

func (ini *IniFile) GetAll(sectionNames ...string) map[string]string {
	m := make(map[string]string)

	if len(sectionNames) == 0 {
		sectionNames = []string{DEFAULT_SECTION}
	}

	keys := ini.Keys(sectionNames...)

	for _, key := range keys {
		value := ""

		for _, sectionName := range sectionNames {
			v, ok := ini.keyValues[sectionName][key]
			if ok {
				value = v
			}
		}

		m[key] = value
	}

	return m
}

func (ini *IniFile) Set(key string, value string, sectionNames ...string) {
	if len(sectionNames) == 0 {
		sectionNames = []string{DEFAULT_SECTION}
	}

	for _, sectionName := range sectionNames {
		keyValues, ok := ini.keyValues[sectionName]
		if !ok {
			ini.keyValues[sectionName] = make(map[string]string)

			keyValues = ini.keyValues[sectionName]
		}

		keyValues[key] = value
	}

	return
}

func (ini *IniFile) Remove(key string, sectionNames ...string) {
	if len(sectionNames) == 0 {
		sectionNames = []string{DEFAULT_SECTION}
	}

	for _, sectionName := range sectionNames {
		keyValues, ok := ini.keyValues[sectionName]
		if ok {
			delete(keyValues, key)
		}

		if len(keyValues) == 0 {
			delete(ini.keyValues, sectionName)
		}
	}

	return
}

func (ini *IniFile) Clear() {
	ini.keyValues = make(map[string]map[string]string)
}

func (ini *IniFile) RemoveSection(sectionNames ...string) {
	if len(sectionNames) == 0 {
		sectionNames = []string{DEFAULT_SECTION}
	}

	for _, sectionName := range sectionNames {
		delete(ini.keyValues, sectionName)
	}
}

func (ini *IniFile) Save() []byte {
	buf := bytes.Buffer{}

	for _, sectionName := range ini.Sections() {
		buf.WriteString(fmt.Sprintf("[%s]\n", sectionName))

		keys := ini.Keys(sectionName)
		keyValues := ini.keyValues[sectionName]

		for _, key := range keys {
			value, ok := keyValues[key]
			if ok {
				if strings.Contains(value, "\n") {
					buf.WriteString(fmt.Sprintf("%s=`\n%s`\n", key, value))
				} else {
					buf.WriteString(fmt.Sprintf("%s=%s\n", key, value))
				}
			}
		}
	}

	return buf.Bytes()
}

func (ini *IniFile) SaveToFile(filename string) error {
	err := os.WriteFile(filename, ini.Save(), DefaultFileMode)
	if Error(err) {
		return err
	}

	return nil
}
