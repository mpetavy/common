package common

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type EventConfigurationReset struct {
	Cfg *bytes.Buffer
}

type EventConfigurationChanged struct {
	Cfg *bytes.Buffer
}

type EventConfigurationLoaded struct {
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
	config      []byte
	configTime  time.Time
)

func init() {
	file = flag.String("cfg.file", CleanPath(AppFilename(".json")), "Configuration file")
	reset = flag.Bool("cfg.reset", false, "Reset configuration file")
	timeout = flag.Int("cfg.timeout", 1000, "rescan timeout for configuration change")
}

func initConfiguration() error {
	DebugFunc()

	err := readEnv()
	if err != nil {
		return err
	}

	ba, ti, err := readFile()
	if err != nil {
		return err
	}

	if *reset {
		err := setFlags(ba)
		if err != nil {
			return err
		}

		ba, ti, err = resetCfg()
		if err != nil {
			return err
		}
	}

	err = activateCfg(ba, ti)
	if err != nil {
		return err
	}

	if *file != "" && *timeout > 0 {
		fileChecker = time.NewTicker(MsecToDuration(*timeout))
		go func() {
			for !AppDeath().IsSet() {
				select {
				case <-fileChecker.C:
					checkChanged()
				}
			}
		}()
	}

	return nil
}

func ResetConfiguration() error {
	_, _, err := resetCfg()

	return err
}

func GetConfiguration() []byte {
	return config
}

func SetConfiguration(ba []byte) error {
	ti, err := writeFile(ba)
	if err != nil {
		return err
	}

	return activateCfg(ba, ti)
}

func activateCfg(ba []byte, ti time.Time) error {
	DebugFunc()

	config = ba
	configTime = ti

	err := setFlags(config)
	if err != nil {
		return err
	}

	Events.Emit(EventConfigurationChanged{bytes.NewBuffer(config)})

	return nil
}

func readFile() ([]byte, time.Time, error) {
	DebugFunc(*file)

	b, err := FileExists(*file)
	if err != nil {
		return nil, time.Time{}, err
	}

	if !b {
		return nil, time.Time{}, nil
	}

	ba, err := ioutil.ReadFile(*file)
	if err != nil {
		return nil, time.Time{}, err
	}

	fileInfo, err := os.Stat(*file)
	if err != nil {
		return nil, time.Time{}, err
	}

	Events.Emit(EventConfigurationLoaded{bytes.NewBuffer(ba)})

	return ba, fileInfo.ModTime(), nil
}

func writeFile(ba []byte) (time.Time, error) {
	DebugFunc(*file)

	buf := bytes.Buffer{}

	err := json.Indent(&buf, ba, "", "    ")
	if err != nil {
		return time.Time{}, err
	}

	if string(buf.Bytes()) != string(config) {
		Debug("Reformat of configuration file done")

		err = ioutil.WriteFile(*file, buf.Bytes(), FileMode(true, true, false))
		if err != nil {
			return time.Time{}, err
		}
	}

	fileInfo, err := os.Stat(*file)
	if err != nil {
		return time.Time{}, err
	}

	return fileInfo.ModTime(), nil
}

func setFlags(ba []byte) error {
	DebugFunc()

	if ba == nil {
		return nil
	}

	cfg := Configuration{}

	err := json.Unmarshal(ba, &cfg)
	if err != nil {
		return err
	}

	if cfg.Flags != nil {
		for k, v := range cfg.Flags {
			p := strings.Index(k, "@")
			if p != -1 {
				targetOs := k[p+1:]
				k = k[:p]

				if targetOs != runtime.GOOS {
					continue
				}
			}

			fl := flag.Lookup(k)
			if fl != nil {
				Debug("flag from %s: %s = %+v", filepath.Base(*file), k, v)

				var err error

				switch value := v.(type) {
				case string:
					err = flag.Set(k, value)
				case float64:
					i := fmt.Sprintf("%.0f", value)
					err = flag.Set(k, i)
				case bool:
					i := strconv.FormatBool(value)
					err = flag.Set(k, i)
				default:
					err = fmt.Errorf("unknown flag type: %+v", value)
				}

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func checkChanged() {
	fi, err := os.Stat(*file)
	if err != nil {
		return
	}

	if fi.ModTime() != configTime {
		DebugFunc()

		ba, ti, err := readFile()
		if err != nil {
			return
		}

		err = activateCfg(ba, ti)
		if err != nil {
			return
		}
	}
}

func readEnv() error {
	flag.VisitAll(func(f *flag.Flag) {
		v := os.Getenv(fmt.Sprintf("%s.%s", Title(), f.Name))

		if v != "" {
			fl := flag.Lookup(f.Name)
			if fl != nil {
				Debug("flag from env: %s = %s", f.Name, v)
				err := flag.Set(f.Name, v)
				if err != nil {
					DebugError(err)
				}
			}
		}
	})

	return nil
}

func resetCfg() ([]byte, time.Time, error) {
	DebugFunc(*file)

	*reset = false

	cfg := Configuration{}
	cfg.Flags = make(map[string]interface{})
	flag.VisitAll(func(fl *flag.Flag) {
		if fl.Value.String() != "" && fl.Value.String() != fl.DefValue {
			cfg.Flags[fl.Name] = fl.Value
		}
	})

	ba, err := json.Marshal(&cfg)
	if err != nil {
		return nil, time.Time{}, err
	}

	buf := bytes.NewBuffer(ba)

	Events.Emit(EventConfigurationReset{buf})

	ti, err := writeFile(buf.Bytes())
	if err != nil {
		return nil, time.Time{}, err
	}

	return buf.Bytes(), ti, nil
}
