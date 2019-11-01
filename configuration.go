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

type EventConfigurationChanged struct {
	Cfg *bytes.Buffer
}

type Configuration struct {
	Flags map[string]interface{} `json:"flags"`
}

var (
	reset   *bool
	file    *string
	timeout *int

	fileChecker *time.Ticker
	fileInfo    os.FileInfo
	fileConfig  []byte

	mapFlag map[string]string
	mapEnv  map[string]string
	mapFile map[string]string
)

func init() {
	file = flag.String("cfg.file", CleanPath(AppFilename(".json")), "Configuration file")
	reset = flag.Bool("cfg.reset", false, "Reset configuration file")
	timeout = flag.Int("cfg.timeout", 1000, "rescan timeout for configuration change")

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
		return err
	}

	err = setFlags()
	if Error(err) {
		return err
	}

	if *reset {
		err = ResetConfiguration()
		if Error(err) {
			return err
		}
	}

	if *file != "" && *timeout > 0 {
		fileChecker = time.NewTicker(MsecToDuration(*timeout))
		go func() {
			for !AppDeath().IsSet() {
				select {
				case <-fileChecker.C:
					Error(checkChanged())
				}
			}
		}()
	}

	return nil
}

func ResetConfiguration() error {
	*reset = false

	cfg := Configuration{}
	cfg.Flags = make(map[string]interface{})
	for k, v := range mapFile {
		cfg.Flags[k] = v
	}

	ba, err := json.Marshal(&cfg)
	if Error(err) {
		return err
	}

	buf := bytes.NewBuffer(ba)

	Events.Emit(EventConfigurationReset{buf})

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

func GetConfiguration() []byte {
	return fileConfig
}

func SetConfiguration(ba []byte) error {
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
	DebugFunc(*file)

	b, err := FileExists(*file)
	if Error(err) {
		return nil, err
	}

	if !b {
		return nil, nil
	}

	ba, err := ioutil.ReadFile(*file)
	if Error(err) {
		return nil, err
	}

	fileConfig = ba

	fileInfo, err = os.Stat(*file)
	if Error(err) {
		return nil, err
	}

	return ba, nil
}

func writeFile(ba []byte) error {
	DebugFunc(*file)

	buf := bytes.Buffer{}

	err := json.Indent(&buf, ba, "", "    ")
	if Error(err) {
		return err
	}

	if string(buf.Bytes()) != string(fileConfig) {
		Debug("Reformat of configuration file done")

		err = ioutil.WriteFile(*file, buf.Bytes(), FileMode(true, true, false))
		if Error(err) {
			return err
		}

		fileInfo, err = os.Stat(*file)
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

			tempErr := flag.Set(f.Name, value)
			if Error(tempErr) && err == nil {
				err = tempErr
			}
		}
	})

	if changed {
		Events.Emit(EventConfigurationChanged{bytes.NewBuffer(fileConfig)})
	}

	return err
}

func checkChanged() error {
	fi, _ := os.Stat(*file)
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
	DebugFunc(*file)

	mapFlag = make(map[string]string)

	flag.Visit(func(f *flag.Flag) {
		mapFlag[f.Name] = f.Value.String()
	})

	return nil
}

func registerEnvFlags() error {
	DebugFunc(*file)

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
	DebugFunc(*file)

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
