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

type configuration struct {
	json         []byte
	jsonFilename string
	jsonFileInfo os.FileInfo
}

type OnConfigurationChangedFunc func(old, new []byte)

var (
	config  configuration
	timeout *int
	checker *time.Ticker

	configurationChangedFunc OnConfigurationChangedFunc
)

func init() {
	timeout = flag.Int("configuration.timeout", 1000, "rescan timeout for configuration change")
}

func initConfiguration() error {
	DebugFunc()

	err := config.readEnv()
	if err != nil {
		return err
	}

	err = config.readJsonFile()
	if err != nil {
		return err
	}

	if config.jsonFilename != "" && *timeout > 0 {
		checker = time.NewTicker(MsecToDuration(*timeout))
		go func() {
			for !AppStopped() {
				select {
				case <-checker.C:
					config.checkChanged()
				}
			}
		}()
	}

	return nil
}

func (this *configuration) GetConfioguration() []byte {
	return this.json
}

func (this *configuration) getFilepath() string {
	path, err := filepath.Abs(".")
	if err != nil {
		path = "."
	}

	path = CleanPath(path)
	if !IsWindowsOS() && !service.Interactive() {
		path = filepath.Join("etc", AppFilename(".log"))
	}

	return path
}

func (this *configuration) readJsonFile() error {
	this.jsonFilename = filepath.Join(this.getFilepath(), AppFilename(".json"))

	b, err := FileExists(this.jsonFilename)
	if err != nil {
		return err
	}

	if !b {
		return nil
	}

	DebugFunc(this.jsonFilename)

	ba, err := ioutil.ReadFile(this.jsonFilename)
	if err != nil {
		return err
	}

	ba = []byte(RemoveJsonComments(string(ba)))

	j, err := NewJason(string(ba))
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
				Debug("flag from %s: %s = %+v", filepath.Base(this.jsonFilename), k, v)

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

	buf := bytes.Buffer{}

	err = json.Indent(&buf, ba, "", "    ")
	if err != nil {
		return err
	}

	if string(buf.Bytes()) != string(ba) {
		err = ioutil.WriteFile(this.jsonFilename, buf.Bytes(), os.ModePerm)
		if err != nil {
			return err
		}

		ba = buf.Bytes()
	}

	this.json = ba
	this.jsonFileInfo, err = os.Stat(this.jsonFilename)
	if err != nil {
		return err
	}

	return nil
}

func (this *configuration) checkChanged() {
	fileInfo, err := os.Stat(config.jsonFilename)
	if err != nil {
		return
	}

	if this.jsonFileInfo != nil && fileInfo.ModTime() != this.jsonFileInfo.ModTime() {
		old := this.json

		err := this.readJsonFile()
		if err != nil {
			Error(err)

			return
		}

		if configurationChangedFunc != nil {
			configurationChangedFunc(old, config.json)
		}
	}
}

func (this *configuration) readEnv() error {
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

func OnConfigurationChange(f OnConfigurationChangedFunc) {
	configurationChangedFunc = f
}
