package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type EventConfigurationReset struct {
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
		FlagCfgReset = systemFlagBool(FlagNameCfgReset, false, "Reset configuration file")
		FlagCfgCreate = systemFlagBool(FlagNameCfgCreate, false, "Reset configuration file and exit")
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

	if !FileExists(*FlagCfgFile) {
		return nil, nil
	}

	DebugFunc(*FlagCfgFile)

	ba, err := os.ReadFile(*FlagCfgFile)
	if Error(err) {
		return nil, err
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

	if len(flagInfos) == 0 {
		mapDefaults, err := registerDefaultFlags()
		if Error(err) {
			return err
		}
		mapCfgFile, err := registerCfgFileFlags()
		if Error(err) {
			return err
		}
		mapIniFile, err := registerIniFileFlags()
		if Error(err) {
			return err
		}
		mapArgs, err := registerArgsFlags()
		if Error(err) {
			return err
		}
		mapEnv, err := registerEnvFlags()
		if Error(err) {
			return err
		}

		maps := []struct {
			origin string
			m      map[string]string
		}{
			{
				origin: "default",
				m:      mapDefaults,
			},
			{
				origin: "ini file",
				m:      mapIniFile,
			},
			{
				origin: "json file",
				m:      mapCfgFile,
			},
			{
				origin: "env",
				m:      mapEnv,
			},
			{
				origin: "args",
				m:      mapArgs,
			},
		}

		flag.VisitAll(func(f *flag.Flag) {
			if IsCmdlineOnlyFlag(f.Name) {
				delete(mapEnv, f.Name)
				delete(mapCfgFile, f.Name)
				delete(mapIniFile, f.Name)
			}

			var origin string
			var value string

			for _, m := range maps {
				v, ok := m.m[f.Name]

				if !ok {
					continue
				}

				origin = m.origin
				value = v
			}

			// ignore GO's test flags
			if !strings.HasPrefix(f.Name, "test.") {
				Error(flag.Set(f.Name, value))
			}

			flagInfos[f.Name] = flagInfo{
				Value:  value,
				Origin: origin,
			}
		})
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

			st.AddCols(f.Name, HidePasswordValue(f.Name, flagValue.Value), IsCmdlineOnlyFlag(f.Name), flagValue.Origin)
		})
	})

	st.Debug()
}

func registerDefaultFlags() (map[string]string, error) {
	DebugFunc()

	m := make(map[string]string)

	flag.VisitAll(func(f *flag.Flag) {
		m[f.Name] = f.Value.String()
	})

	return m, nil
}

func registerArgsFlags() (map[string]string, error) {
	DebugFunc()

	m := make(map[string]string)

	flag.Visit(func(f *flag.Flag) {
		if IsFlagProvided(f.Name) {
			m[f.Name] = f.Value.String()
		}
	})

	return m, nil
}

func registerEnvFlags() (map[string]string, error) {
	DebugFunc()

	m := make(map[string]string)

	flag.VisitAll(func(f *flag.Flag) {
		envName := fmt.Sprintf("%s_%s", Title(), strings.ReplaceAll(f.Name, ".", "_"))

		envValue := os.Getenv(strings.ToUpper(envName))
		if envValue == "" {
			envValue = os.Getenv(strings.ToLower(envName))
		}

		if envValue != "" {
			m[f.Name] = envValue
		}
	})

	return m, nil
}

func registerCfgFileFlags() (map[string]string, error) {
	DebugFunc(*FlagCfgFile)

	m := make(map[string]string)

	cfg, err := LoadConfigurationFile[Configuration]()
	if Error(err) {
		return nil, err
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
