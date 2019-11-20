package common

import (
	"bufio"
	"bytes"
	"cloud.google.com/go/translate"
	"context"
	"flag"
	"fmt"
	"github.com/fatih/structtag"
	"github.com/go-ini/ini"
	googlelanguage "golang.org/x/text/language"
	"google.golang.org/api/option"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

var (
	language       *string
	systemLanguage string
	i18nFile       *ini.File
)

const (
	DEFAULT_LANGUAGE = "en"
)

func init() {
	language = flag.String("language", "", "language for messages")
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

func initLanguage() {
	filename := AppFilename(fmt.Sprintf(".i18n"))
	ba := GetResource(filename)

	if ba != nil {
		var err error

		i18nFile, err = ini.Load(ba)
		if Error(err) {
			return
		}

		lang := *language
		if lang == "" {
			var err error

			lang, err = GetSystemLanguage()
			if Error(err) {
				return
			}
		}

		if lang != "" {
			WarnError(SetLanguage(lang))
		}
	}

	DebugError(fmt.Errorf("language file %s does not exist", filename))
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

	sec, _ := i18nFile.GetSection(lang)
	if sec == nil {
		return fmt.Errorf("language %s is not available", lang)
	}

	*language = lang

	return nil
}

//GetLanguages lists all available languages
func GetLanguages() ([]string, error) {
	list := make([]string, 0)

	if i18nFile != nil {
		for _, sec := range i18nFile.Sections() {
			if sec.Name() != ini.DefaultSection {
				list = append(list, sec.Name())
			}
		}
	} else {
		list = append(list, DEFAULT_LANGUAGE)
	}

	return list, nil
}

// Translate a message to the current set language
func Translate(msg string, args ...interface{}) string {
	if i18nFile != nil && *language != "" {
		sec, _ := i18nFile.GetSection(*language)
		if sec != nil {
			m, err := sec.GetKey(msg)

			if err == nil && m.Value() != "" {
				msg = m.Value()
			}
		}
	}

	return fmt.Sprintf(msg, args...)
}

func googleTranslate(googleApiKey string, text string, foreignLanguage string) (string, error) {
	DebugFunc("%s -> %s ...", text, foreignLanguage)

	ctx := context.Background()

	// Creates a client.
	client, err := translate.NewClient(ctx, option.WithAPIKey(googleApiKey))
	if err != nil {
		return "", fmt.Errorf("Failed to create client: %v", err)
	}

	// Sets the target language.
	target, err := googlelanguage.Parse(foreignLanguage)
	if err != nil {
		return "", fmt.Errorf("Failed to parse target language: %v", err)
	}

	// Translates the text into Russian.
	translations, err := client.Translate(ctx, []string{text}, target, nil)
	if err != nil {
		return "", fmt.Errorf("Failed to translate text: %v", err)
	}

	DebugFunc("%s -> %s: %s", text, foreignLanguage, translations[0].Text)

	return translations[0].Text, nil
}

func CreateI18nFile(objs ...interface{}) error {
	googleApiKey, ok := os.LookupEnv("GOOGLE_API_KEY")
	if !ok {
		return fmt.Errorf("Failed to get Google API key from env: GOOGLE_API_KEY")
	}

	filename := filepath.Join("static", AppFilename(fmt.Sprintf(".i18n")))

	i18ns := make([]string, 0)

	//get i18n from source

	regexTranslate := regexp.MustCompile("Translate\\(\"(.*?)\"")
	regexSubstitution := regexp.MustCompile("\\%[^v]")

	err := WalkFilepath("*.go", true, func(path string) error {
		Debug("extract i18n from source file: %s", path)

		ba, err := ioutil.ReadFile(path)
		if Error(err) {
			return err
		}

		findings := regexTranslate.FindAll(ba, -1)
		if findings == nil {
			return nil
		}

		for _, f := range findings {
			finding := string(f)

			finding = finding[strings.Index(finding, "\"")+1 : len(finding)-1]

			if regexSubstitution.Match([]byte(finding)) {
				return fmt.Errorf("invalid substitution: %s", finding)
			}

			i18ns = append(i18ns, finding)
		}

		return nil
	})
	if Error(err) {
		return err
	}

	for _, obj := range objs {
		err = scanStruct(&i18ns, obj)
		if Error(err) {
			return err
		}
	}

	sort.Strings(i18ns)

	// remove duplicates

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

	// remove non existing i18ns from list

	for _, sec := range i18nFile.Sections() {
		if sec.Name() != ini.DefaultSection {
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
	}

	// update all languages with found i18ns

	secNames := []string{DEFAULT_LANGUAGE, "zh-Google", "fr-Google"}
	for _, sec := range i18nFile.Sections() {
		if sec.Name() != ini.DefaultSection {
			index, _ := IndexOf(sec, secNames)
			if index == -1 {
				secNames = append(secNames, sec.Name())
			}
		}
	}

	for _, i18n := range i18ns {
		for _, secName := range secNames {
			foreignLanguage := secName
			p := strings.Index(foreignLanguage, "-")
			if p != -1 {
				foreignLanguage = foreignLanguage[:p]
			}

			sec, _ := i18nFile.GetSection(secName)
			if sec == nil {
				sec, _ = i18nFile.NewSection(secName)
			}

			key, _ := sec.GetKey(i18n)
			value := ""

			if key != nil {
				value = key.Value()
			}

			if value == "" {
				if secName == DEFAULT_LANGUAGE {
					value = i18n
				} else {
					var err error

					value, err = googleTranslate(googleApiKey, strings.ReplaceAll(i18n, "%v", "XXX"), foreignLanguage)
					if WarnError(err) {
						value = ""
					} else {
						value = strings.ReplaceAll(value, "XXX", "%v")
					}
				}
			}

			if key == nil {
				_, err := i18nFile.Section(sec.Name()).NewKey(i18n, value)
				if Error(err) {
					return err
				}
			} else {
				i18nFile.Section(sec.Name()).Key(i18n).SetValue(value)
			}
		}
	}

	sortedFile := ini.Empty()

	for _, sec := range i18nFile.Sections() {
		if sec.Name() != ini.DefaultSection {
			keys := sec.KeyStrings()

			sort.Strings(keys)

			newSec, err := sortedFile.NewSection(sec.Name())
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
	}

	i18nFile = sortedFile

	Debug("update i18n file: %s", filename)

	err = i18nFile.SaveTo(filename)
	if Error(err) {
		return err
	}

	return nil
}
