package common

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

type EventConfigurationReset struct {
	Cfg *bytes.Buffer
}

type Configuration struct {
	Flags map[string]string `json:"flags"`
}

func (this *Configuration) SetFlag(flagName string, flagValue string) error {
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

func NewConfiguration() *Configuration {
	cfg := Configuration{}
	cfg.Flags = make(map[string]string)

	return &cfg
}

var (
	FlagCfgReset   *bool
	FlagCfgFile    *string
	FlagCfgTimeout *int

	fileChecker *time.Ticker
	fileInfo    os.FileInfo
	fileConfig  []byte

	mapFlag map[string]string
	mapEnv  map[string]string
	mapFile map[string]string
)

func init() {
	FlagCfgFile = flag.String("cfg.file", CleanPath(AppFilename(".json")), "Configuration file")
	FlagCfgReset = flag.Bool("cfg.reset", false, "Reset configuration file")
	FlagCfgTimeout = flag.Int("cfg.timeout", 0, "rescan timeout for configuration change") // FIXME

	mapFlag = make(map[string]string)
	mapEnv = make(map[string]string)
	mapFile = make(map[string]string)
}

func initConfiguration() error {
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

	err = registerFileFlags(ba)
	if Error(err) {
		if IsRunningAsService() {
			*FlagCfgReset = true
		} else {
			return err
		}
	}

	err = setFlags()
	if Error(err) {
		return err
	}

	if *FlagCfgReset {
		err = ResetConfiguration()
		if Error(err) {
			return err
		}
	}

	if *FlagCfgFile != "" && *FlagCfgTimeout > 0 {
		fileChecker = time.NewTicker(MsecToDuration(*FlagCfgTimeout))
		go func() {
			for AppLifecycle().IsSet() {
				select {
				case <-fileChecker.C:
					WarnError(checkChanged())
				}
			}
		}()
	}

	return nil
}

func ResetConfiguration() error {
	*FlagCfgReset = false

	cfg := NewConfiguration()

	for k, v := range mapFile {
		cfg.Flags[k] = v
	}

	ba, err := json.Marshal(cfg)
	if Error(err) {
		return err
	}

	buf := bytes.NewBuffer(ba)

	Events.Emit(EventConfigurationReset{buf})

	err = writeFile(buf.Bytes())
	if Error(err) {
		return err
	}

	err = registerFileFlags(buf.Bytes())
	if Error(err) {
		return err
	}

	err = setFlags()
	if Error(err) {
		return err
	}

	return nil
}

func GetConfiguration() *Configuration {
	DebugFunc()

	ba := GetConfigurationBuffer()

	if ba == nil {
		return nil
	}

	cfg := Configuration{}

	err := json.Unmarshal(ba, &cfg)
	if Error(err) {
		return nil
	}

	return &cfg
}

func GetConfigurationBuffer() []byte {
	return fileConfig
}

func SetConfigurationBuffer(ba []byte) error {
	err := writeFile(ba)
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

	b, err := FileExists(*FlagCfgFile)
	if Error(err) {
		return nil, err
	}

	if !b {
		return nil, nil
	}

	ba, err := ioutil.ReadFile(*FlagCfgFile)
	if Error(err) {
		return nil, err
	}

	fileConfig = ba

	fileInfo, err = os.Stat(*FlagCfgFile)
	if Error(err) {
		return nil, err
	}

	return ba, nil
}

func writeFile(ba []byte) error {
	DebugFunc(*FlagCfgFile)

	buf := bytes.Buffer{}

	err := json.Indent(&buf, ba, "", "    ")
	if Error(err) {
		return err
	}

	if string(buf.Bytes()) != string(fileConfig) {
		Debug("Reformat of configuration file done")

		Error(FileBackup(*FlagCfgFile))

		err = ioutil.WriteFile(*FlagCfgFile, buf.Bytes(), DefaultFileMode)
		if Error(err) {
			return err
		}

		fileInfo, err = os.Stat(*FlagCfgFile)
		if Error(err) {
			return err
		}
	}

	fileConfig = buf.Bytes()

	return nil
}

func setFlags() error {
	DebugFunc()

	var err error

	changed := false

	flag.VisitAll(func(f *flag.Flag) {
		if f.Name == "cfg.reset" {
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

		if value != "" && value != f.Value.String() {
			changed = true

			Debug("Set flag %s : %s [%s]", f.Name, value, origin)

			WarnError(flag.Set(f.Name, value))
		}
	})

	return err
}

func checkChanged() error {
	fi, _ := os.Stat(*FlagCfgFile)
	if fi == nil {
		return nil
	}

	if fi.ModTime() != fileInfo.ModTime() {
		DebugFunc()

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
	}

	return nil
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
	DebugFunc(*FlagCfgFile)

	mapEnv = make(map[string]string)

	flag.VisitAll(func(f *flag.Flag) {
		v := os.Getenv(fmt.Sprintf("%s.%s", Title(), f.Name))

		if v != "" {
			mapEnv[f.Name] = v
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
