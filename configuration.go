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
	ApplicationTitle   string            `json:"applicationTitle"`
	ApplicationVersion string            `json:"applicationVersion"`
	Flags              map[string]string `json:"flags"`
}

var (
	FlagCfgReset *bool
	FlagCfgFile  *string

	mapFlag = make(map[string]string)
	mapEnv  = make(map[string]string)
	mapFile = make(map[string]string)
)

func init() {
	FlagCfgFile = flag.String("cfg.file", CleanPath(AppFilename(".json")), "Configuration file")
	FlagCfgReset = flag.Bool("cfg.reset", false, "Reset configuration file")
}

func NewConfiguration() *Configuration {
	cfg := Configuration{}

	cfg.ApplicationTitle = Title()
	cfg.ApplicationVersion = Version(true, true, true)
	cfg.Flags = make(map[string]string)

	return &cfg
}

func (this *Configuration) SetFlag(flagName string, flagValue string) error {
	if IsOneTimeFlag(flagName) {
		return nil
	}

	if flag.Lookup(flagName) == nil {
		return fmt.Errorf("unknown flag: %s", flagName)
	}

	this.Flags[flagName] = flagValue

	return nil
}

func (this *Configuration) GetFlag(flagName string) (string, error) {
	if flag.Lookup(flagName) == nil {
		return "", fmt.Errorf("unknown flag: %s", flagName)
	}

	return this.Flags[flagName], nil
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

	ba, err := readFile()
	if Error(err) {
		return err
	}

	*FlagCfgReset = *FlagCfgReset || ba == nil

	err = registerFileFlags(ba)
	if Error(err) {
		return err
	}

	err = setFlags(false)
	if Error(err) {
		return err
	}

	// only respect settings from os.Flags and os.Env once, after that only from file

	Events.Emit(EventFlagsSet{})

	if *FlagCfgReset {
		err = ResetConfiguration()
		if Error(err) {
			return err
		}
	}

	return nil
}

func ResetConfiguration() error {
	DebugFunc()

	*FlagCfgReset = false

	buf := &bytes.Buffer{}

	if Events.Emit(EventConfigurationReset{buf}) {
		err := writeFile(buf.Bytes())
		if Error(err) {
			return err
		}
	}

	err := registerFileFlags(buf.Bytes())
	if Error(err) {
		return err
	}

	err = setFlags(true)
	if Error(err) {
		return err
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

	err = setFlags(false)
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

	err := json.Indent(&buf, ba, "", "    ")
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

	ba, err = json.MarshalIndent(m, "", "    ")
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

func setFlags(reset bool) error {
	DebugFunc()

	var err error

	flag.VisitAll(func(f *flag.Flag) {
		if IsOneTimeFlag(f.Name) {
			return
		}

		vFlag, bFlag := mapFlag[f.Name]
		vEnv, bEnv := mapEnv[f.Name]
		vFile, bFile := mapFile[f.Name]

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

		if value == "" && reset {
			value = f.DefValue
		}

		if value != "" && value != f.Value.String() {
			Debug("Set flag %s : %s [%s]", f.Name, value, origin)

			Error(flag.Set(f.Name, value))
		}
	})

	return err
}

func registerArgsFlags() error {
	DebugFunc(*FlagCfgFile)

	mapFlag = make(map[string]string)

	flag.Visit(func(f *flag.Flag) {
		mapFlag[f.Name] = f.Value.String()
	})

	return nil
}

func registerEnvFlags() error {
	DebugFunc()

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
		for k, v := range cfg.Flags {
			value := fmt.Sprintf("%v", v)

			mapFile[k] = value
		}
	}

	return nil
}
