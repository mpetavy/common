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
	Cfg *bytes.Buffer
}

type Configuration struct {
	ApplicationTitle   string    `json:"applicationTitle"`
	ApplicationVersion string    `json:"applicationVersion"`
	Flags              KeyValues `json:"flags"`
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
		FlagCfgFile = flag.String(FlagNameCfgFile, "", "Configuration file")
		FlagCfgReset = flag.Bool(FlagNameCfgReset, false, "Reset configuration file")
		FlagCfgCreate = flag.Bool(FlagNameCfgCreate, false, "Reset configuration file and exit")
	})

	Events.AddListener(EventFlagsParsed{}, func(event Event) {
		if *FlagCfgFile == "" {
			*FlagCfgFile = CleanPath(AppFilename(".json"))
		}
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

	buf := &bytes.Buffer{}

	Events.Emit(EventConfigurationReset{buf}, false)

	if buf.Len() > 0 {
		err := SaveConfigurationFile(buf.Bytes())
		if Error(err) {
			return err
		}
	}

	return nil
}

func LoadConfiguration() (*Configuration, error) {
	DebugFunc()

	cfg := NewConfiguration()

	ba, err := LoadConfigurationFile()
	if Error(err) {
		return nil, err
	}

	if ba == nil {
		return cfg, nil
	}

	err = json.Unmarshal(ba, cfg)
	if Error(err) {
		return nil, err
	}

	return cfg, nil
}

func SaveConfiguration(cfg interface{}) error {
	ba, err := json.MarshalIndent(cfg, "", "  ")
	if Error(err) {
		return err
	}

	err = SaveConfigurationFile(ba)
	if Error(err) {
		return err
	}

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

func LoadConfigurationFile() ([]byte, error) {
	DebugFunc()

	if !FileExists(*FlagCfgFile) {
		exe, err := os.Executable()
		if Error(err) {
			return nil, err
		}

		f := CleanPath(filepath.Join(filepath.Dir(exe), filepath.Base(*FlagCfgFile)))

		if !FileExists(f) {
			return nil, nil
		}

		err = flag.Set(FlagNameCfgFile, f)
		if Error(err) {
			return nil, err
		}
	}

	DebugFunc(*FlagCfgFile)

	ba, err := os.ReadFile(*FlagCfgFile)
	if Error(err) {
		return nil, err
	}

	return []byte(RemoveJsonComments(string(ba))), nil
}

func SaveConfigurationFile(ba []byte) error {
	DebugFunc(*FlagCfgFile)

	buf := bytes.Buffer{}

	err := json.Indent(&buf, ba, "", "  ")
	if Error(err) {
		return err
	}

	m := make(map[string]interface{})

	err = json.Unmarshal(buf.Bytes(), &m)
	if Error(err) {
		return err
	}

	if v, ok := m["applicationTitle"]; !ok || v == "" {
		m["applicationTitle"] = Title()
	}

	if v, ok := m["applicationVersion"]; !ok || v == "" {
		m["applicationVersion"] = Version(true, true, true)
	}

	ba, err = json.MarshalIndent(m, "", "  ")
	if Error(err) {
		return err
	}

	buf.Reset()
	_, err = buf.Write(ba)
	if Error(err) {
		return err
	}

	fileConfig, err := LoadConfigurationFile()
	if Error(err) {
		return err
	}

	if string(buf.Bytes()) != string(fileConfig) {
		Debug("Reformat of configuration file %s done", *FlagCfgFile)

		Error(FileBackup(*FlagCfgFile))

		err = os.WriteFile(*FlagCfgFile, buf.Bytes(), DefaultFileMode)
		if Error(err) {
			return err
		}
	}

	return nil
}

func setFlags() error {
	DebugFunc()

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
			origin: "env",
			m:      mapEnv,
		},
		{
			origin: "cfg file",
			m:      mapCfgFile,
		},
		{
			origin: "ini file",
			m:      mapIniFile,
		},
		{
			origin: "args",
			m:      mapArgs,
		},
	}

	flag.VisitAll(func(f *flag.Flag) {
		if IsCmdlineOnlyFlag(f.Name) {
			return
		}

		var origin string
		var value string

		for _, m := range maps {
			v, ok := m.m[f.Name]

			if !ok || v == "" {
				continue
			}

			origin = m.origin
			value = v
		}

		if value != "" && value != f.Value.String() {
			Debug("Set flag %s : %s [%s]", f.Name, value, origin)

			Error(flag.Set(f.Name, value))
		}
	})

	flag.VisitAll(func(fl *flag.Flag) {
		v := fmt.Sprintf("%+v", fl.Value)
		if strings.Contains(strings.ToLower(fl.Name), "password") || strings.Contains(strings.ToLower(fl.Name), "pwd") {
			v = strings.Repeat("X", len(v))
		}

		Debug("flag %s = %+v", fl.Name, v)
	})

	Events.Emit(EventFlagsSet{}, false)

	return err
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
		envName := strings.ReplaceAll(fmt.Sprintf("%s.%s", Title(), f.Name), ".", "_")
		envValue := strings.ToLower(os.Getenv(envName))
		if envValue == "" {
			envValue = strings.ToUpper(os.Getenv(envName))
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

	ba, err := LoadConfigurationFile()
	if Error(err) {
		return m, err
	}

	if ba == nil {
		return m, nil
	}

	cfg := Configuration{}

	err = json.Unmarshal(ba, &cfg)
	if Error(err) {
		return m, err
	}

	if cfg.Flags != nil {
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
