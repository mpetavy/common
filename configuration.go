package common

import (
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

func init() {
	FlagCfgFile = flag.String(FlagNameCfgFile, "", "Configuration file")
	FlagCfgReset = flag.Bool(FlagNameCfgReset, false, "Reset configuration file")
	FlagCfgCreate = flag.Bool(FlagNameCfgCreate, false, "Reset configuration file and exit")

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
		return fmt.Errorf("unknown flag: %s", flagName)
	}

	err := this.Flags.Put(flagName, flagValue)
	if Error(err) {
		return err
	}

	return nil
}

func (this *Configuration) GetFlag(flagName string) (string, error) {
	if flag.Lookup(flagName) == nil {
		return "", fmt.Errorf("unknown flag: %s", flagName)
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

	ba, err := LoadConfigurationFile()
	if Error(err) {
		return err
	}

	err = SetFlags(ba)
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

	ba, err := LoadConfigurationFile()
	if Error(err) {
		return nil, err
	}

	if ba == nil {
		return nil, nil
	}

	cfg := NewConfiguration()

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

func LoadConfigurationFile() ([]byte, error) {
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

func getValue(m map[string]string, key string) (string, bool) {
	if m == nil {
		return "", false
	}

	v, ok := m[key]

	return v, ok
}

func SetFlags(ba []byte) error {
	DebugFunc()

	mapFlag, err := registerArgsFlags()
	mapEnv, err := registerEnvFlags()
	mapFile, err := registerFileFlags(ba)

	flag.VisitAll(func(f *flag.Flag) {
		if IsCmdlineOnlyFlag(f.Name) {
			return
		}

		vFlag, bFlag := getValue(mapFlag, f.Name)
		vEnv, bEnv := getValue(mapEnv, f.Name)
		vFile, bFile := getValue(mapFile, f.Name)

		value := ""
		origin := ""

		if bFile {
			value = vFile
			origin = "file"
		}

		if bEnv {
			value = vEnv
			origin = "env"
		}
		if bFlag {
			value = vFlag
			origin = "flag"
		}

		if value != "" && value != f.Value.String() {
			Debug("Set flag %s : %s [%s]", f.Name, value, origin)

			Error(flag.Set(f.Name, value))
		}
	})

	flag.VisitAll(func(fl *flag.Flag) {
		v := fmt.Sprintf("%+v", fl.Value)
		if strings.Contains(strings.ToLower(fl.Name), "password") {
			v = strings.Repeat("X", len(v))
		}

		Debug("flag %s = %+v", fl.Name, v)
	})

	Events.Emit(EventFlagsSet{}, false)

	return err
}

// IsFlagSetOnArgs reports back all really set flags on the command line.
// Go's flag.Visit() gives back also the flags which have been set before by "flag.Set(..."
func IsFlagSetOnArgs(fn string) bool {
	fnSingle := "-" + fn
	fnEqual := "-" + fn + "="

	for _, f := range os.Args {
		if f == fnSingle || strings.HasPrefix(f, fnEqual) {
			return true
		}
	}

	return false
}

func registerArgsFlags() (map[string]string, error) {
	DebugFunc()

	m := make(map[string]string)

	flag.Visit(func(f *flag.Flag) {
		if IsFlagSetOnArgs(f.Name) {
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

func registerFileFlags(ba []byte) (map[string]string, error) {
	DebugFunc(*FlagCfgFile)

	m := make(map[string]string)

	if ba == nil {
		return m, nil
	}

	cfg := Configuration{}

	err := json.Unmarshal(ba, &cfg)
	if Error(err) {
		return m, err
	}

	if cfg.Flags != nil {
		for _, key := range cfg.Flags.Keys() {
			value, _ := cfg.Flags.Get(key)

			m[key] = value
		}
	}

	return m, nil
}
