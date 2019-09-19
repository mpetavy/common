package common

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kardianos/service"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type EventConfigurationChanged struct {
	old, new *[]byte
}

type EventConfigurationLoaded struct {
	cfg *[]byte
}

type Configuration struct {
	Flags map[string]interface{} `json:"flags"`
}

var (
	reset   *bool
	file    *string
	timeout *int
	checker *time.Ticker

	fileInfo os.FileInfo
	config   []byte
)

func init() {
	path := CleanPath(AppFilename(".json"))
	if !IsWindowsOS() && !service.Interactive() {
		path = filepath.Join("/etc", AppFilename(".json"))
	}

	file = flag.String("cfg.file", path, "Configuration file")
	reset = flag.Bool("cfg.reset", false, "Reset configuration file")
	timeout = flag.Int("cfg.timeout", 1000, "rescan timeout for configuration change")
}

func initConfiguration() error {
	DebugFunc()

	err := readEnv()
	if err != nil {
		return err
	}

	err = readFile()
	if err != nil {
		return err
	}

	err = setFlags()
	if err != nil {
		return err
	}

	if *reset {
		err := resetConfiguration()
		if err != nil {
			return err
		}

		err = SetConfiguration(config)
		if err != nil {
			return err
		}
	}

	if *file != "" && *timeout > 0 {
		checker = time.NewTicker(MsecToDuration(*timeout))
		go func() {
			for !AppStopped() {
				select {
				case <-checker.C:
					checkChanged()
				}
			}
		}()
	}

	return nil
}

func GetConfiguration() []byte {
	DebugFunc()

	return config
}

func SetConfiguration(cfg []byte) error {
	DebugFunc()

	old := config

	buf := bytes.Buffer{}

	err := json.Indent(&buf, cfg, "", "    ")
	if err != nil {
		return err
	}

	if string(buf.Bytes()) != string(config) {
		err = ioutil.WriteFile(*file, buf.Bytes(), FileMode(true, true, false))
		if err != nil {
			return err
		}

		config = buf.Bytes()

		err = setFlags()
		if err != nil {
			return err
		}

		Events.Emit(EventConfigurationChanged{&old, &config})
	}

	return nil
}

func readFile() error {
	DebugFunc(file)

	b, err := FileExists(*file)
	if err != nil {
		return err
	}

	if !b {
		return nil
	}

	config, err = ioutil.ReadFile(*file)
	if err != nil {
		return err
	}

	fileInfo, err = os.Stat(*file)
	if err != nil {
		return err
	}

	Events.Emit(EventConfigurationLoaded{&config})

	return nil
}

func setFlags() error {
	DebugFunc()

	if config == nil {
		return nil
	}

	cfg := Configuration{}

	err := json.Unmarshal(config, &cfg)
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
	if fileInfo != nil {
		fi, err := os.Stat(*file)
		if err != nil {
			return
		}

		if fi.ModTime() != fileInfo.ModTime() {
			err := readFile()
			if err != nil {
				Error(err)

				return
			}

			err = SetConfiguration(config)
			if err != nil {
				Error(err)

				return
			}
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

func resetConfiguration() error {
	DebugFunc(*file)

	*reset = false

	cfg := Configuration{Flags: make(map[string]interface{})}

	flag.VisitAll(func(fl *flag.Flag) {
		if fl.Value.String() != "" && fl.Value.String() != fl.DefValue {
			cfg.Flags[fl.Name] = fl.Value
		}
	})

	var err error

	config, err = json.Marshal(&cfg)
	if err != nil {
		return err
	}

	return nil
}
