package common

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

type EventConfigurationReset struct {
	Cfg *bytes.Buffer
}

type Configuration struct {
	ApplicationTitle   string       `json:"applicationTitle"`
	ApplicationVersion string       `json:"applicationVersion"`
	Flags              KeyValueList `json:"flags"`
}

var (
	FlagCfgReset *bool
	FlagCfgFile  *string

	mapFlag map[string]string
	mapEnv  map[string]string
	mapFile map[string]string
)

func init() {
	FlagCfgFile = flag.String("cfg.file", CleanPath(AppFilename(".json")), "Configuration file")
	FlagCfgReset = flag.Bool("cfg.reset", false, "Reset configuration file")
}

func NewConfiguration() *Configuration {
	cfg := Configuration{}

	cfg.ApplicationTitle = Title()
	cfg.ApplicationVersion = Version(true, true, true)

	return &cfg
}

func (this *Configuration) SetFlag(flagName string, flagValue string) error {
	if IsOneTimeFlag(flagName) {
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

func IsOneTimeFlag(n string) bool {
	list := []string{
		"cfg.reset",
		"test",
	}

	for _, l := range list {
		if strings.HasPrefix(n, l) {
			return true
		}

	}

	return false
}

func InitConfiguration() error {
	DebugFunc()

	err := registerArgsFlags()
	if Error(err) {
		return err
	}

	err = registerEnvFlags()
	if Error(err) {
		return err
	}

	*FlagCfgReset = *FlagCfgReset || !FileExists(*FlagCfgFile)

	if *FlagCfgReset {
		*FlagCfgReset = false

		err = ResetConfiguration()
		if Error(err) {
			return err
		}
	}

	ba, err := readFile()
	if Error(err) {
		return err
	}

	err = registerFileFlags(ba)
	if Error(err) {
		return err
	}

	err = setFlags()
	if Error(err) {
		return err
	}

	return nil
}

func ResetConfiguration() error {
	DebugFunc()

	buf := &bytes.Buffer{}

	if Events.Emit(EventConfigurationReset{buf}) && buf.Len() > 0 {
		err := writeFile(buf.Bytes())
		if Error(err) {
			return err
		}
	}

	return nil
}

func GetConfiguration() (*Configuration, error) {
	DebugFunc()

	ba, err := GetConfigurationBuffer()
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

func GetConfigurationBuffer() ([]byte, error) {
	return readFile()
}

func SetConfiguration(cfg interface{}) error {
	ba, err := json.MarshalIndent(cfg, "", "  ")
	if Error(err) {
		return err
	}

	s := string(ba)
	s = strings.Replace(s, "\"application\": \"\",", fmt.Sprintf("\"application\": \"%s\",", TitleVersion(true, true, true)), 1)

	ba = []byte(s)

	err = writeFile(ba)
	if Error(err) {
		return err
	}

	err = registerFileFlags(ba)
	if Error(err) {
		return err
	}

	err = setFlags()
	if Error(err) {
		return err
	}

	return nil
}

func readFile() ([]byte, error) {
	DebugFunc(*FlagCfgFile)

	if !FileExists(*FlagCfgFile) {
		return nil, nil
	}

	ba, err := os.ReadFile(*FlagCfgFile)
	if Error(err) {
		return nil, err
	}

	return []byte(RemoveJsonComments(string(ba))), nil
}

func writeFile(ba []byte) error {
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

	fileConfig, err := readFile()
	if Error(err) {
		return err
	}

	if string(buf.Bytes()) != string(fileConfig) {
		Debug("Reformat of configuration file done")

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

func setFlags() error {
	DebugFunc()

	var err error

	flag.VisitAll(func(f *flag.Flag) {
		if IsOneTimeFlag(f.Name) {
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

	if err == nil {
		Events.Emit(EventFlagsSet{})
	}

	return err
}

func registerArgsFlags() error {
	DebugFunc(*FlagCfgFile)

	// Golang bug, by using "flag.Set" the original command line flags are falsely extended
	if mapFlag != nil {
		return nil
	}

	mapFlag = make(map[string]string)

	flag.Visit(func(f *flag.Flag) {
		mapFlag[f.Name] = f.Value.String()
	})

	return nil
}

func registerEnvFlags() error {
	DebugFunc()

	if mapEnv != nil {
		return nil
	}

	mapEnv = make(map[string]string)

	flag.VisitAll(func(f *flag.Flag) {
		envName := strings.ReplaceAll(fmt.Sprintf("%s.%s", Title(), f.Name), ".", "_")
		envValue := strings.ToLower(os.Getenv(envName))
		if envValue == "" {
			envValue = strings.ToUpper(os.Getenv(envName))
		}

		if envValue != "" {
			mapEnv[f.Name] = envValue
		}
	})

	return nil
}

func registerFileFlags(ba []byte) error {
	DebugFunc(*FlagCfgFile)

	mapFile = make(map[string]string)

	if ba == nil {
		return nil
	}

	cfg := Configuration{}

	err := json.Unmarshal(ba, &cfg)
	if Error(err) {
		return err
	}

	if cfg.Flags != nil {
		for _, key := range cfg.Flags.Keys() {
			value, _ := cfg.Flags.Get(key)

			mapFile[key] = value
		}
	}

	return nil
}
