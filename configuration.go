package common

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
)

type EventConfigurationReset struct {
}

type EventFlagsExternal struct {
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
	FlagCfgReset  *bool
	FlagCfgCreate *bool
	FlagCfgFile   *string

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
	}

	flagInfos = make(map[string]flagInfo)
)

const (
	FlagNameCfgFile   = "cfg.file"
	FlagNameCfgReset  = "cfg.reset"
	FlagNameCfgCreate = "cfg.create"
)

type ErrUnknownFlag struct {
	Origin string
	Name   string
}

func (e *ErrUnknownFlag) Error() string {
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

		FlagCfgFile = flag.String(FlagNameCfgFile, CleanPath(filepath.Join(dir, AppFilename(".json"))), "Configuration file")
		FlagCfgReset = SystemFlagBool(FlagNameCfgReset, false, "Reset configuration file")
		FlagCfgCreate = SystemFlagBool(FlagNameCfgCreate, false, "Reset configuration file and exit")
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
			b, _ := EqualWildcards(flagName, mask)
			if b {
				r = true

				break
			}
		}
	}

	DebugFunc("%s: %v", flagName, r)

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

			os.Exit(0)
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

	m := make(map[string]string)

	f := CleanPath(AppFilename(".ini"))
	if !FileExists(f) {
		return m, nil
	}

	ba, err := os.ReadFile(f)
	if Error(err) {
		return m, err
	}

	if !bytes.HasSuffix(ba, []byte("\n")) {
		ba = append(ba, '\n')
	}

	withCrlf, err := NewSeparatorSplitFunc(nil, []byte("\n"), false)
	if Error(err) {
		return m, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(ba))
	scanner.Split(withCrlf)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		p := strings.Index(line, "=")
		if p == -1 {
			continue
		}

		key := strings.TrimSpace(line[:p])
		value := strings.TrimSpace(line[p+1:])

		if value == "`" {
			sb := strings.Builder{}

			for scanner.Scan() {
				line = scanner.Text()
				if strings.HasPrefix(line, "`") {
					break
				}

				sb.WriteString(line)
			}

			value = sb.String()
		}

		if strings.HasPrefix(value, "@") {
			ba, err := os.ReadFile(value[1:])
			if Error(err) {
				return m, err
			}

			value = string(ba)
		}

		if flag.Lookup(key) == nil {
			return m, &ErrUnknownFlag{
				Origin: filepath.Base(f),
				Name:   key,
			}
		}

		m[key] = value
	}

	return m, nil
}

func LoadConfigurationFile[T any]() (*T, error) {
	DebugFunc()

	var ba []byte
	var err error

	filenameOrJson := strings.TrimSpace(*FlagCfgFile)

	if filenameOrJson != "" && strings.HasPrefix(filenameOrJson, "{") {
		Debug("Read flag %s as JSON content", FlagNameCfgFile)

		ba = []byte(filenameOrJson)

		if strings.HasPrefix("base64:", filenameOrJson) {
			ba, err = base64.StdEncoding.DecodeString(filenameOrJson[7:])
			if Error(err) {
				return nil, err
			}
		}
	} else {
		Debug("Read flag %s as JSON file: %scontent", FlagNameCfgFile, filenameOrJson)

		if !FileExists(filenameOrJson) {
			return nil, &ErrFileNotFound{
				FileName: filenameOrJson,
			}
		}

		ba, err = os.ReadFile(filenameOrJson)
		if Error(err) {
			return nil, err
		}
	}

	ba, err = RemoveJsonComments(ba)
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

	// is flag.Set(...=) ist used then the correct list fo cmdline flags is destroyed, that's why here preserved...
	argsFlags, _ := registerArgsFlags()

	flagMaps := []struct {
		origin     string
		fn         func() (map[string]string, error)
		flags      map[string]string
		initialSet bool
	}{
		{
			origin:     "default",
			fn:         registerDefaultFlags,
			initialSet: false,
		},
		{
			origin:     "ini file",
			fn:         registerIniFileFlags,
			initialSet: true,
		},
		{
			origin:     "cfg file",
			fn:         registerCfgFileFlags,
			initialSet: true,
		},
		{
			origin:     "env",
			fn:         registerEnvFlags,
			initialSet: true,
		},
		{
			origin:     "args",
			flags:      argsFlags,
			initialSet: true,
		},
		{
			origin:     "external",
			fn:         registerExternalFlags,
			initialSet: true,
		},
		{
			origin:     "args",
			fn:         nil,
			flags:      argsFlags,
			initialSet: true,
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

		if flagMaps[i].initialSet {
			for key, value := range flagMaps[i].flags {
				if IsCmdlineOnlyFlag(key) {
					continue
				}

				if strings.HasPrefix(value, "$ENV(") && strings.HasSuffix(value, ")") {
					envName := value[5 : len(value)-1]
					envValue, ok := os.LookupEnv(envName)
					if !ok {
						return fmt.Errorf("ENV variable cannot be evaluated: %s", value)
					}

					value = envValue
				}

				err := flag.Set(key, value)
				if Error(err) {
					return err
				}

				flagInfos[key] = flagInfo{
					Value:  value,
					Origin: flagMaps[i].origin,
				}
			}
		}
	}

	debugFlags()

	Events.Emit(EventFlagsSet{}, false)

	return nil
}

func debugFlags() {
	st := NewStringTable()
	st.AddCols("Flag", "Value", "Only cmdline", "Origin")

	NoDebug(func() {
		flag.VisitAll(func(f *flag.Flag) {
			flagValue := flagInfos[f.Name]

			st.AddCols(f.Name, HidePasswordValue(f.Name, flagValue.Value), fmt.Sprintf("%v", IsCmdlineOnlyFlag(f.Name)), flagValue.Origin)
		})
	})

	st.Debug()
}

func registerDefaultFlags() (map[string]string, error) {
	DebugFunc()

	m := make(map[string]string)

	flag.VisitAll(func(f *flag.Flag) {
		if IsCmdlineOnlyFlag(f.Name) {
			return
		}

		m[f.Name] = f.Value.String()
	})

	return m, nil
}

func registerArgsFlags() (map[string]string, error) {
	DebugFunc()

	m := make(map[string]string)

	flag.Visit(func(f *flag.Flag) {
		m[f.Name] = f.Value.String()
	})

	return m, nil
}

func registerExternalFlags() (map[string]string, error) {
	DebugFunc()

	m := make(map[string]string)

	event := &EventFlagsExternal{}
	event.Flags = make(map[string]string)

	Events.Emit(event, false)

	for key, value := range event.Flags {
		if flag.Lookup(key) == nil {
			return nil, &ErrUnknownFlag{
				Origin: "External",
				Name:   key,
			}
		}

		m[key] = value
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
		if IsCmdlineOnlyFlag(f.Name) {
			return
		}

		value := os.Getenv(FlagNameAsEnvName(f.Name))

		if value != "" {
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
				return m, &ErrUnknownFlag{
					Origin: *FlagCfgFile,
					Name:   key,
				}
			}

			m[key] = value
		}
	}

	return m, nil
}
func MandatoryFlags(excludes ...string) []string {
	excludes = append(excludes, "test*")

	mandatoryFlags := []string{}

	isExcluded := func(flagName string) bool {
		for _, exclude := range excludes {
			b, _ := EqualWildcards(flagName, exclude)

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

				if IsFlagProvided(flagName) || IsFlagAsEnvProvided(flagName) || fl.Value.String() != "" || fl.DefValue != "" {
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
