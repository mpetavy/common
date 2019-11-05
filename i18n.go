package common

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/fatih/structtag"
	"github.com/go-ini/ini"
	"io"
	"io/ioutil"
	"os/exec"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	Language       *string
	systemLanguage string
	i18nFile       *ini.File
)

func init() {
	Language = flag.String("language", "", "language for messages")
}

//GetSystemLanguage return BCP 47 standard language name
func GetSystemLanguage() (string, error) {
	DebugFunc()

	if IsWindowsOS() {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd := exec.Command("powershell", "Get-WinSystemLocale")

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := Watchdog(cmd, time.Second*3)
		if Error(err) {
			return "", err
		}

		output := string(stdout.Bytes())

		r := bufio.NewReader(strings.NewReader(output))

		b := false

		for {
			line, err := r.ReadString('\n')
			if err == io.EOF {
				break
			}

			if b {
				p := strings.Index(line, " ")

				systemLanguage = strings.TrimSpace(line[p:])
				p = strings.Index(systemLanguage, " ")
				systemLanguage = strings.TrimSpace(systemLanguage[:p])

				p = strings.Index(systemLanguage, "-")
				if p != -1 {
					systemLanguage = systemLanguage[:p]
				}

				DebugFunc(systemLanguage)

				return systemLanguage, nil
			} else {
				b = strings.HasPrefix(line, "----")
			}
		}
	} else {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd := exec.Command("locale")

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := Watchdog(cmd, time.Second)
		if Error(err) {
			return "", err
		}

		output := strings.TrimSpace(string(stdout.Bytes()))

		r := bufio.NewReader(strings.NewReader(output))

		for {
			line, err := r.ReadString('\n')
			if err == io.EOF {
				break
			}

			if strings.HasPrefix(line, "LANGUAGE=") {
				systemLanguage := strings.TrimSpace(line[strings.Index(line, "=")+1:])

				p := strings.Index(systemLanguage, "_")
				if p != -1 {
					systemLanguage = systemLanguage[:p]
				}

				DebugFunc(systemLanguage)

				return systemLanguage, nil
			}
		}
	}

	return "", fmt.Errorf("cannot get system language")
}

func initLanguage() error {
	filename := AppFilename(fmt.Sprintf(".i18n"))

	b, _ := FileExists(filename)

	if b {
		var err error

		i18nFile, err = ini.Load(filename)
		if Error(err) {
			return err
		}

		if *Language == "" {
			*Language, err = GetSystemLanguage()
			if Error(err) {
				return err
			}
		}

		if *Language != "" {
			Error(SetLanguage(*Language))
		}

		return nil
	}

	return fmt.Errorf("language file %s does not exist", filename)
}

func scanStruct(i18ns *[]string, data interface{}) error {
	if reflect.TypeOf(data).Kind() == reflect.Ptr {
		data = reflect.ValueOf(data).Elem()
	}

	structValue := reflect.ValueOf(data)

	for i := 0; i < structValue.NumField(); i++ {
		fieldType := structValue.Type().Field(i)
		fieldValue := structValue.Field(i)

		if fieldType.Type.Kind() == reflect.Struct {
			err := scanStruct(i18ns, fieldValue.Interface())
			if Error(err) {
				return err
			}
		}

		fieldTags, err := structtag.Parse(string(fieldType.Tag))
		if err != nil {
			return err
		}

		tag, err := fieldTags.Get("html")
		if err != nil {
			continue
		}

		for _, i18n := range *i18ns {
			if i18n == tag.Name {
				continue
			}
		}

		*i18ns = append(*i18ns, tag.Name)
	}

	return nil
}

//SetLanguage sets the language file to translation
func SetLanguage(lang string) error {
	DebugFunc(lang)

	if i18nFile == nil {
		return fmt.Errorf("no language file available")
	}

	*Language = lang

	return nil
}

//GetLanguages lists all available languages
func GetLanguages() ([]string, error) {
	list := make([]string, 0)

	secs := i18nFile.Sections()
	for _, sec := range secs {
		list = append(list, sec.Name())
	}

	return list, nil
}

// Translate a message to the current set language
func Translate(msg string, args ...interface{}) string {
	if i18nFile != nil && *Language != "" {
		sec, _ := i18nFile.GetSection(*Language)
		if sec != nil {
			m, err := sec.GetKey(msg)

			if err == nil && m.Value() != "" {
				msg = m.Value()
			}
		}
	}

	return fmt.Sprintf(msg, args...)
}

func CreateI18nFile(objs ...interface{}) error {
	i18ns := make([]string, 0)

	re := regexp.MustCompile("Translate\\(\"(.*?)\"")

	err := WalkFilepath("*.go", true, func(path string) error {
		Debug("extract i18n from source file: %s", path)

		ba, err := ioutil.ReadFile(path)
		if Error(err) {
			return err
		}

		findings := re.FindAll(ba, -1)
		if findings == nil {
			return nil
		}

		for _, f := range findings {
			finding := string(f)
			finding = finding[strings.Index(finding, "\"")+1 : len(finding)-1]

			i18ns = append(i18ns, finding)
		}

		return nil
	})

	for _, obj := range objs {
		err = scanStruct(&i18ns, obj)
		if Error(err) {
			return err
		}
	}

	sort.Strings(i18ns)

	for i := 1; i < len(i18ns); i++ {
		if i18ns[i] == i18ns[i-1] {
			if i+1 == len(i18ns) {
				i18ns = i18ns[0 : len(i18ns)-1]
			} else {
				i18ns = append(i18ns[:i], i18ns[i+1:]...)
			}
		}
	}

	if i18nFile == nil {
		i18nFile = ini.Empty()
	}

	for _, sec := range i18nFile.Sections() {
		keys := sec.KeyStrings()

		for _, key := range keys {
			found := false

			for _, i18n := range i18ns {
				found = i18n == key
				if found {
					break
				}
			}

			if !found {
				sec.DeleteKey(key)
			}
		}
	}

	for _, i18n := range i18ns {
		secs := i18nFile.Sections()

		for _, sec := range secs {
			key, _ := sec.GetKey(i18n)
			if key != nil {
				if sec.Name() == ini.DefaultSection {
					key.SetValue(i18n)
				}
			} else {
				if sec.Name() == ini.DefaultSection {
					Ignore(sec.NewKey(i18n, i18n))
				} else {
					Ignore(sec.NewKey(i18n, ""))
				}
			}
		}
	}

	newFile := ini.Empty()

	for _, sec := range i18nFile.Sections() {
		keys := sec.KeyStrings()

		sort.Strings(keys)

		newSec, err := newFile.NewSection(sec.Name())
		if Error(err) {
			return err
		}

		for _, key := range keys {
			k, err := sec.GetKey(key)
			if Error(err) {
				return err
			}

			Ignore(newSec.NewKey(key, k.Value()))
		}
	}

	i18nFile = newFile

	Debug("update i18n file: %s", AppFilename(fmt.Sprintf(".i18n")))

	err = i18nFile.SaveTo(AppFilename(fmt.Sprintf(".i18n")))
	if Error(err) {
		return err
	}

	return nil
}
