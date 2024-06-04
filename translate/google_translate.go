package translate

import (
	googleTranslate "cloud.google.com/go/translate"
	"context"
	"fmt"
	"github.com/fatih/structtag"
	"github.com/go-ini/ini"
	"github.com/mpetavy/common"
	googlelanguage "golang.org/x/text/language"
	"google.golang.org/api/option"
	"html"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

func scanStruct(i18ns *[]string, data interface{}) error {
	return common.IterateStruct(data, func(fieldPath string, fieldType reflect.StructField, fieldValue reflect.Value) error {
		fieldTags, err := structtag.Parse(string(fieldType.Tag))
		if common.Error(err) {
			return err
		}

		tagHtml, err := fieldTags.Get("html")
		if err != nil {
			return nil
		}

		for _, i18n := range *i18ns {
			if i18n == tagHtml.Name {
				continue
			}
		}

		*i18ns = append(*i18ns, tagHtml.Name)

		return nil
	})
}

func GoogleTranslate(googleApiKey string, text string, foreignLanguage string) (string, error) {
	common.DebugFunc("%s -> %s ...", text, foreignLanguage)

	ctx := context.Background()

	// Creates a client.
	client, err := googleTranslate.NewClient(ctx, option.WithAPIKey(googleApiKey))
	if common.Error(err) {
		return "", err
	}

	source, err := googlelanguage.Parse("en")
	if common.Error(err) {
		return "", err
	}

	// Sets the target language.
	target, err := googlelanguage.Parse(foreignLanguage)
	if common.Error(err) {
		return "", err
	}

	// Translates the text
	translations, err := client.Translate(ctx, []string{text}, target, &googleTranslate.Options{
		Source: source,
		Format: "",
		Model:  "",
	})
	if common.Error(err) {
		return "", err
	}

	term := html.UnescapeString(translations[0].Text)

	if strings.Index(foreignLanguage, "-") == -1 {
		term = strings.ReplaceAll(term, "-", " ")
	}
	term = strings.ReplaceAll(term, " / ", "/")

	common.DebugFunc("%s -> %s: %s", text, foreignLanguage, term)

	return term, nil
}

func CreateI18nFile(I18nFile *ini.File, path string, objs ...interface{}) error {
	if path == "" {
		return nil
	}

	googleApiKey, ok := os.LookupEnv("GOOGLE_API_KEY")
	if !ok {
		return fmt.Errorf("Failed to get Google API key from env: GOOGLE_API_KEY")
	}

	filename := filepath.Join(path, common.AppFilename(".i18n"))

	i18ns := make([]string, 0)

	//get i18n from source

	rxs := []*regexp.Regexp{regexp.MustCompile("Translate\\(\"(.*?)\""), regexp.MustCompile("TranslateFor\\(.*\"(.*?)\"")}
	regexSubstitution := regexp.MustCompile("\\%[^v]")

	paths := []string{"*.go", "../common/*.go"}

	for _, path := range paths {
		err := common.WalkFiles(path, true, false, func(path string, f os.FileInfo) error {
			if f.IsDir() {
				return nil
			}

			common.Debug("extract i18n from source file: %s", path)

			ba, err := os.ReadFile(path)
			if common.Error(err) {
				return err
			}

			for _, regexTranslate := range rxs {
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
			}

			return nil
		})
		if common.Error(err) {
			return err
		}
	}

	for _, obj := range objs {
		err := scanStruct(&i18ns, obj)
		if common.Error(err) {
			return err
		}
	}

	common.SortStringsCaseInsensitive(i18ns)

	// remove duplicates

	r, err := regexp.Compile("\\%[^v%]")
	if common.Error(err) {
		return err
	}

	for i := 1; i < len(i18ns); i++ {
		if r.Match([]byte(i18ns[i])) {
			return fmt.Errorf("invalid substitution parameter found: %s", i18ns[i])
		}

		if i18ns[i] == i18ns[i-1] {
			if i+1 == len(i18ns) {
				i18ns = i18ns[0 : len(i18ns)-1]
			} else {
				i18ns = common.SliceDelete(i18ns, i)
				i--
			}
		}
	}

	common.SortStringsCaseInsensitive(i18ns)

	if I18nFile == nil {
		I18nFile = ini.Empty()
	}

	// remove non existing i18ns from list

	for _, sec := range I18nFile.Sections() {
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

	secNames := []string{common.GetDefaultLanguage(), "nl", "it", "el", "es", "ar", "zh", "fr", "th", "de"}
	for _, sec := range I18nFile.Sections() {
		if sec.Name() != ini.DefaultSection {
			if common.IndexOf(secNames, sec) == -1 {
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

			sec, _ := I18nFile.GetSection(secName)
			if sec == nil {
				sec, _ = I18nFile.NewSection(secName)
			}

			key, _ := sec.GetKey(i18n)
			value := ""

			if key != nil {
				value = key.String()
			}

			if value == "" {
				if secName == common.GetDefaultLanguage() {
					value = i18n
				} else {
					var err error

					value, err = GoogleTranslate(googleApiKey, strings.ReplaceAll(i18n, "%v", "XXX"), foreignLanguage)
					if common.Error(err) {
						value = ""
					} else {
						value = strings.ReplaceAll(value, "XXX", "%v")
					}
				}
			}

			if key == nil {
				_, err := I18nFile.Section(sec.Name()).NewKey(i18n, value)
				if common.Error(err) {
					return err
				}
			} else {
				I18nFile.Section(sec.Name()).Key(i18n).SetValue(value)
			}
		}
	}

	sortedFile := ini.Empty()

	for _, sec := range I18nFile.Sections() {
		if sec.Name() != ini.DefaultSection {
			keys := sec.KeyStrings()

			common.SortStringsCaseInsensitive(keys)

			newSec, err := sortedFile.NewSection(sec.Name())
			if common.Error(err) {
				return err
			}

			for _, key := range keys {
				k, err := sec.GetKey(key)
				if common.Error(err) {
					return err
				}

				_, err = newSec.NewKey(key, k.String())
				if common.Error(err) {
					return err
				}
			}
		}
	}

	I18nFile = sortedFile

	common.Debug("update i18n file: %s", filename)

	err = I18nFile.SaveTo(filename)
	if common.Error(err) {
		return err
	}

	return nil
}
