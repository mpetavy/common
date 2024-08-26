package common

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/go-ini/ini"
	"io"
	"os/exec"
	"strings"
	"time"
)

var (
	FlagAppLanguage *string
	systemLanguage  string
	I18nFile        *ini.File
)

const (
	FlagNameAppLanguage = "app.language"
)

func init() {
	Events.AddListener(EventInit{}, func(ev Event) {
		FlagAppLanguage = systemFlagString(FlagNameAppLanguage, "en", "language for messages")
	})

	Events.AddListener(EventFlagsSet{}, func(ev Event) {
		initLanguage()
	})
}

func GetDefaultLanguage() string {
	return flag.Lookup(FlagNameAppLanguage).DefValue
}

// GetSystemLanguage return BCP 47 standard language name
func GetSystemLanguage() (string, error) {
	DebugFunc()

	if IsWindows() {
		cmd := exec.Command("powershell", "Get-WinSystemLocale")

		ba, err := NewWatchdogCmd(cmd, time.Second*3)
		if Error(err) {
			return "", err
		}

		output := string(ba)

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
		_, err := exec.LookPath("locale")
		if err != nil {
			return "en", nil
		}

		cmd := exec.Command("locale")

		ba, err := NewWatchdogCmd(cmd, time.Second)
		if Error(err) {
			return "", err
		}

		output := strings.TrimSpace(string(ba))

		r := bufio.NewReader(strings.NewReader(output))

		for {
			line, err := r.ReadString('\n')
			if err == io.EOF {
				break
			}

			if strings.HasPrefix(line, "LANGUAGE=") || strings.HasPrefix(line, "LANG=") {
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
	ba, _, _ := ReadResource(AppFilename(".i18n"))

	if ba != nil {
		var err error

		I18nFile, err = ini.Load(ba)
		if DebugError(err) {
			return
		}

		Error(SetLanguage(GetDefaultLanguage()))
	}
}

// SetLanguage sets the language file to translation
func SetLanguage(lang string) error {
	DebugFunc(lang)

	if I18nFile == nil {
		return fmt.Errorf("no language file available")
	}

	sec, _ := I18nFile.GetSection(lang)
	if sec == nil {
		return fmt.Errorf("language %s is not available", lang)
	}

	*FlagAppLanguage = lang

	return nil
}

// GetLanguages lists all available languages
func GetLanguages() ([]string, error) {
	list := make([]string, 0)

	if I18nFile != nil {
		for _, sec := range I18nFile.Sections() {
			if sec.Name() != ini.DefaultSection {
				list = append(list, sec.Name())
			}
		}
	} else {
		list = append(list, GetDefaultLanguage())
	}

	SortStringsCaseInsensitive(list)

	return list, nil
}

func TranslateFor(language, msg string) string {
	if I18nFile != nil && language != "" {
		sec, _ := I18nFile.GetSection(language)
		if sec != nil {
			m, err := sec.GetKey(msg)

			if err == nil && m.Value() != "" {
				return m.Value()
			}
		}
	}

	return msg
}

// Translate a message to the current set language
func Translate(msg string, args ...interface{}) string {
	if msg == "" {
		return ""
	}

	t := TranslateFor(*FlagAppLanguage, msg)

	return Capitalize(fmt.Sprintf(t, args...))
}
