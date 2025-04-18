package common

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
)

type EventConfigurationReset struct {
}

type EventExternalFlags struct {
	Err   error
	Flags map[string]string
}

type Configuration struct {
	ApplicationTitle   string    `json:"applicationTitle"`
	ApplicationVersion string    `json:"applicationVersion"`
	Flags              KeyValues `json:"flags"`
}

type flagInfo struct {
	Value  string
	Origin string
}

var (
	FlagCfgExternal   *string
	FlagCfgReset      *bool
	FlagCfgCreate     *bool
	FlagCfgFile       *string
	FlagCfgIniFile    *string
	FlagCfgIniSection *string

	CmdlineOnlyFlags = []string{
		FlagNameService,
		FlagNameServiceUsername,
		FlagNameServicePassword,
		FlagNameServiceTimeout,
		FlagNameCfgFile,
		FlagNameCfgReset,
		FlagNameCfgCreate,
		FlagNameUsage,
		FlagNameUsageMd,
		FlagNameCfgIniFile,
		FlagNameCfgIniSection,
	}

	flagInfos = make(map[string]flagInfo)
)

const (
	FlagNameCfgFile       = "cfg.file"
	FlagNameCfgExternal   = "cfg.external"
	FlagNameCfgReset      = "cfg.reset"
	FlagNameCfgCreate     = "cfg.create"
	FlagNameCfgIniFile    = "cfg.ini.file"
	FlagNameCfgIniSection = "cfg.ini.section"
)

type ErrUnknownFlag struct {
	Origin string
	Name   string
}

func (e *ErrUnknownFlag) Error() string {
	if e.Origin == "" {
		return fmt.Sprintf("unknown flag: %s", e.Name)
	}

	return fmt.Sprintf("unknown flag in %s: %s", e.Origin, e.Name)
}

func init() {
	Events.AddListener(EventInit{}, func(ev Event) {
		var dir string

		if IsRunningAsService() {
			exe, err := os.Executable()
			Panic(err)

			dir = filepath.Dir(exe)
		} else {
			wd, err := os.Getwd()
			Panic(err)

			dir = wd
		}

		FlagCfgFile = SystemFlagString(FlagNameCfgFile, CleanPath(filepath.Join(dir, AppFilename(".json"))), "JSON file configuration path")
		FlagCfgExternal = SystemFlagString(FlagNameCfgExternal, "", "Configuration JSON content")
		FlagCfgReset = SystemFlagBool(FlagNameCfgReset, false, "Reset configuration file")
		FlagCfgCreate = SystemFlagBool(FlagNameCfgCreate, false, "Reset configuration file and exit")
		FlagCfgIniFile = SystemFlagString(FlagNameCfgIniFile, CleanPath(filepath.Join(dir, AppFilename(".ini"))), "INI file configuration path")
		FlagCfgIniSection = SystemFlagString(FlagNameCfgIniSection, DEFAULT_SECTION, "INI file section")
	})
}

func NewConfiguration() *Configuration {
	cfg := Configuration{}

	cfg.ApplicationTitle = Title()
	cfg.ApplicationVersion = Version(true, true, true)

	return &cfg
}

func (this *Configuration) SetFlag(flagName string, flagValue string) error {
	if IsCmdlineOnlyFlag(flagName) {
		return nil
	}

	if flag.Lookup(flagName) == nil {
		return &ErrUnknownFlag{
			Origin: "configuration",
			Name:   flagName,
		}
	}

	err := this.Flags.Put(flagName, flagValue)
	if Error(err) {
		return err
	}

	return nil
}

func (this *Configuration) GetFlag(flagName string) (string, error) {
	if flag.Lookup(flagName) == nil {
		return "", &ErrUnknownFlag{
			Origin: "configuration",
			Name:   flagName,
		}

	}

	flagValue, _ := this.Flags.Get(flagName)

	return flagValue, nil
}

func IsCmdlineOnlyFlag(flagName string) bool {
	r := false

	for _, mask := range CmdlineOnlyFlags {
		if mask == flagName {
			r = true

			break
		}
	}

	if !r {
		list := []string{
			"test*",
		}

		for _, mask := range list {
			b, _ := EqualsWildcard(flagName, mask)
			if b {
				r = true

				break
			}
		}
	}

	return r
}

func initConfiguration() error {
	DebugFunc()

	*FlagCfgReset = *FlagCfgReset || !FileExists(*FlagCfgFile)

	if *FlagCfgReset || *FlagCfgCreate {
		*FlagCfgReset = false

		if *FlagCfgCreate {
			*FlagIoFileBackups = 0
		}

		err := ResetConfiguration()
		if Error(err) {
			return err
		}

		if *FlagCfgCreate {
			Info("File created: %s", *FlagCfgFile)

			Exit(0)
		}
	}

	err := setFlags()
	if Error(err) {
		return err
	}

	return nil
}

func ResetConfiguration() error {
	DebugFunc()

	Events.Emit(EventConfigurationReset{}, false)

	return nil
}

func registerIniFileFlags() (map[string]string, error) {
	DebugFunc()

	if !FileExists(*FlagCfgIniFile) {
		return nil, nil
	}

	ini := NewIniFile()

	err := ini.LoadFile(*FlagCfgIniFile)
	if Error(err) {
		return nil, err
	}

	m := ini.GetAll(Split(*FlagCfgIniSection, ",")...)

	for key, value := range m {
		if flag.Lookup(key) == nil {
			return nil, &ErrUnknownFlag{
				Origin: "ini file",
				Name:   key,
			}
		}

		if !IsValidFlagDefinition(key, value, true) {
			delete(m, key)
		}
	}

	return m, nil
}

func LoadConfigurationFile[T any]() (*T, error) {
	DebugFunc()

	if *FlagCfgFile == "" {
		return nil, &ErrFileNotFound{}
	}

	var content string

	if FileExists(*FlagCfgFile) {
		DebugFunc("read cfg from file: %s", *FlagCfgFile)

		ba, err := os.ReadFile(*FlagCfgFile)
		if Error(err) {
			return nil, err
		}

		content = string(ba)
	} else {
		fl := flag.Lookup(FlagNameCfgExternal)

		// configuration set the to flag cfg.file=<content of configuration file>

		if fl.Value.String() != fl.DefValue {
			content = fl.Value.String()

			if content != "" {
				DebugFunc("read cfg from flag: %s", FlagNameCfgExternal)
			}
		}
	}

	content = strings.TrimSpace(content)

	if content == "" {
		return nil, &ErrFileNotFound{
			FileName: *FlagCfgFile,
		}
	}

	ba, err := RemoveJsonComments([]byte(content))
	if Error(err) {
		return nil, err
	}

	cfg := new(T)

	err = json.Unmarshal(ba, cfg)
	if Error(err) {
		return nil, err
	}

	return cfg, nil
}

func SaveConfigurationFile(cfg any) error {
	DebugFunc(*FlagCfgFile)

	ba, err := json.MarshalIndent(cfg, "", "  ")
	if Error(err) {
		return err
	}

	m := make(map[string]interface{})

	err = json.Unmarshal(ba, &m)
	if Error(err) {
		return err
	}

	if v, ok := m["applicationTitle"]; !ok || v == "" {
		m["applicationTitle"] = Title()
	}

	if v, ok := m["app"]; !ok || v == "" {
		m["applicationVersion"] = Version(true, true, true)
	}

	ba, err = json.MarshalIndent(m, "", "  ")
	if Error(err) {
		return err
	}

	err = os.WriteFile(*FlagCfgFile, ba, DefaultFileMode)
	if Error(err) {
		return err
	}

	return nil
}

func setFlags() error {
	DebugFunc()

	if len(flagInfos) != 0 {
		Debug("set cached flag values")

		for name, info := range flagInfos {
			fl := flag.Lookup(name)
			if fl != nil {
				err := flag.Set(name, info.Value)
				if Error(err) {
					return err
				}
			}
		}

		return nil
	}

	Debug("create cached flag values")

	defaultFlags, err := registerDefaultFlags()
	if Error(err) {
		return err
	}

	argsFlags, err := registerArgsFlags()
	if Error(err) {
		return err
	}

	envFlags, err := registerEnvFlags()
	if Error(err) {
		return err
	}

	externalCfg := ""

	flagMaps := []struct {
		origin string
		fn     func() (map[string]string, error)
		flags  map[string]string
	}{
		{
			origin: "default",
			flags:  defaultFlags,
		},
		{
			origin: "env",
			flags:  envFlags,
		},
		{
			origin: "args",
			flags:  argsFlags,
		},
		{
			origin: "ini file",
			fn:     registerIniFileFlags,
		},
		{
			origin: "external",
			fn: func() (map[string]string, error) {
				m, err := registerExternalFlags()
				if err == nil {
					var ok bool

					externalCfg, ok = m[FlagNameCfgExternal]
					if !ok {
						externalCfg = ""
					}
				}

				return m, err
			},
		},
		{
			origin: "cfg file",
			fn:     registerCfgFileFlags,
		},
		{
			origin: "ini file",
			fn:     registerIniFileFlags,
		},
		{
			origin: "env",
			flags:  envFlags,
		},
		{
			origin: "args",
			flags:  argsFlags,
		},
	}

	for i := 0; i < len(flagMaps); i++ {
		var err error

		if flagMaps[i].fn != nil {
			flagMaps[i].flags, err = flagMaps[i].fn()
			if Error(err) {
				return err
			}
		}

		for key, value := range flagMaps[i].flags {
			switch {
			case IsEncrypted(value):
				var err error

				value, err = Secret(value)
				if Error(err) {
					return err
				}
			case strings.HasPrefix(value, "env:"):
				envName := value[4:]
				envValue, ok := os.LookupEnv(envName)
				if !ok {
					return fmt.Errorf("ENV variable cannot be evaluated: %s", value)
				}

				value = envValue
			}

			if key == FlagNameCfgExternal {
				value = externalCfg
			}

			err := flag.Set(key, value)
			if Error(err) {
				return errors.Wrap(err, fmt.Sprintf("[%s] %s: %s\n", flagMaps[i].origin, key, value))
			}

			flagInfos[key] = flagInfo{
				Value:  value,
				Origin: flagMaps[i].origin,
			}
		}
	}

	Events.Emit(EventFlags{}, false)

	debugFlags()

	return nil
}

func debugFlags() {
	st := NewStringTable()
	st.AddCols("Flag", "ENV name", "Value", "Origin", "Only cmdline")

	flag.VisitAll(func(f *flag.Flag) {
		flagValue := flagInfos[f.Name]
		flagOrigin := flagValue.Origin
		flagOnlyCmdline := ""

		if IsCmdlineOnlyFlag(f.Name) {
			flagOnlyCmdline = "*"
		}

		value := HideSecretFlags(f.Name, flagValue.Value)

		if f.Name == FlagNameCfgExternal {
			value = CapString(value, 80)
		}

		st.AddCols(f.Name, FlagNameAsEnvName(f.Name), value, flagOrigin, flagOnlyCmdline)
	})

	Debug(fmt.Sprintf("Flags\n%s", st.Table()))
}

func IsValidFlagDefinition(name string, value string, checkCmdlineFlag bool) bool {
	return flag.Lookup(name) != nil &&
		!strings.HasPrefix(name, "test.") &&
		strings.TrimSpace(value) != "" &&
		(!checkCmdlineFlag || !IsCmdlineOnlyFlag(name))
}

func registerDefaultFlags() (map[string]string, error) {
	DebugFunc()

	m := make(map[string]string)

	flag.VisitAll(func(f *flag.Flag) {
		// With "dummy" be use ALL defaultFlags even with an empty value but skip "test.*"

		if IsValidFlagDefinition(f.Name, "dummy", false) {
			m[f.Name] = f.Value.String()
		}
	})

	return m, nil
}

func registerArgsFlags() (map[string]string, error) {
	DebugFunc()

	m := make(map[string]string)

	flag.Visit(func(f *flag.Flag) {
		if IsValidFlagDefinition(f.Name, f.Value.String(), false) {
			m[f.Name] = f.Value.String()
		}
	})

	return m, nil
}

func registerExternalFlags() (map[string]string, error) {
	DebugFunc()

	m := make(map[string]string)

	event := &EventExternalFlags{}
	event.Flags = make(map[string]string)

	Events.Emit(event, false)

	if Error(event.Err) {
		return nil, event.Err
	}

	for key, value := range event.Flags {
		if IsValidFlagDefinition(key, value, true) {
			m[key] = value
		}
	}

	return m, nil
}

func FlagNameAsEnvName(flagName string) string {
	return strings.ToUpper(fmt.Sprintf("%s_%s", Title(), strings.ReplaceAll(flagName, ".", "_")))
}

func registerEnvFlags() (map[string]string, error) {
	DebugFunc()

	m := make(map[string]string)

	flag.VisitAll(func(f *flag.Flag) {
		value := os.Getenv(FlagNameAsEnvName(f.Name))

		if IsValidFlagDefinition(f.Name, value, true) {
			m[f.Name] = value
		}
	})

	return m, nil
}

func registerCfgFileFlags() (map[string]string, error) {
	DebugFunc(*FlagCfgFile)

	m := make(map[string]string)

	cfg, err := LoadConfigurationFile[Configuration]()
	_, ok := err.(*ErrFileNotFound)
	if ok {
		return m, nil
	}
	if Error(err) {
		return m, err
	}

	if cfg != nil && cfg.Flags != nil {
		for _, key := range cfg.Flags.Keys() {
			value, _ := cfg.Flags.Get(key)

			if flag.Lookup(key) == nil {
				return nil, &ErrUnknownFlag{
					Origin: "cfg file",
					Name:   key,
				}
			}

			if IsValidFlagDefinition(key, value, true) {
				m[key] = value
			}
		}
	}

	return m, nil
}
func MandatoryFlags(excludes ...string) []string {
	excludes = append(excludes, "test*")

	mandatoryFlags := []string{}

	isExcluded := func(flagName string) bool {
		for _, exclude := range excludes {
			b, _ := EqualsWildcard(flagName, exclude)

			if b {
				return true
			}
		}

		return false
	}

	flag.VisitAll(func(f *flag.Flag) {
		isZero := reflect.ValueOf(f.Value).Elem().IsZero()

		if !slices.Contains(SystemFlagNames, f.Name) && isZero && f.DefValue == "" && !isExcluded(f.Name) {
			mandatoryFlags = append(mandatoryFlags, f.Name)
		}
	})

	return mandatoryFlags
}

func checkMandatoryFlags(flags []string) error {
	if *FlagService == SERVICE_UNINSTALL {
		return nil
	}

	if flags != nil {
		notDefined := strings.Builder{}

		for _, mf := range flags {
			choices := strings.Split(mf, "|")
			for i := 0; i < len(choices); i++ {
				choices[i] = "\"-" + choices[i] + "\""
			}

			allChoices := strings.Join(choices, " or ")
			defined := 0

			for _, flagName := range strings.Split(mf, "|") {
				fl := flag.Lookup(flagName)

				if fl == nil || fl.Value == nil {
					return fmt.Errorf("unknown mandatory flag: \"%s\"", flagName)
				}

				if IsFlagProvided(flagName) || IsFlagAsEnvProvided(flagName) || fl.Value.String() != fl.DefValue {
					defined++
				}
			}

			switch {
			case defined == 0:
				if notDefined.Len() > 0 {
					notDefined.WriteString(", ")
				}
				notDefined.WriteString(allChoices)
			case defined > 1:
				return TraceError(fmt.Errorf("only one mandatory flag allowed: %s", allChoices))
			}
		}

		if notDefined.Len() > 0 {
			return TraceError(fmt.Errorf("mandatory flag not defined: %s", notDefined.String()))
		}
	}

	return nil
}
