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

type Cfg struct {
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
	if *reset {
		err := resetConfiguration()
		if err != nil {
			return err
		}
	}

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

		Events.Emit(EventConfigurationChanged{&old, &config})
	}

	err = setFlags()
	if err != nil {
		return err
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

	return SetConfiguration(config)
}

func setFlags() error {
	DebugFunc()

	if config == nil {
		return nil
	}
	j, err := NewJason(string(config))
	if err != nil {
		return err
	}

	if j.Exists("flags") {
		j, err = j.Element("flags")
		if err != nil {
			return err
		}

		for k, v := range j.attributes {
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

			Events.Emit(EventConfigurationLoaded{&config})
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

	return FileDelete(*file)
}
