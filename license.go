package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type require struct {
	Name        string
	Version     string
	Comment     string
	LicenseName string
	LicenseUrl  string
	LicenseText string
}

type gomodInfo struct {
	Module  string
	Version string
	Require []require
}

func URLGet(url string) ([]byte, error) {
	h := &http.Client{}

	username := os.Getenv("GITHUB_USERNAME")
	password := os.Getenv("GITHUB_PASSWORD")

	if username == "" || password == "" {
		return nil, fmt.Errorf("Failed to get GITHUB credentials API key from env: GITHUB_USERNAME, GITHUB_PASSWORD")
	}

	req, err := http.NewRequest("GET", url, nil)
	if Error(err) {
		return nil, err
	}

	req.SetBasicAuth(username, password)

	var r *http.Response

	r, err = h.Do(req)
	if Error(err) {
		return nil, err
	}

	ba, err := ioutil.ReadAll(r.Body)

	defer func() {
		Error(r.Body.Close())
	}()

	if Error(err) {
		return nil, err
	}

	return ba, nil
}

func CreateModLicenseFile(path string) error {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return fmt.Errorf("undefined GOPATH")
	}

	filename := "go.mod"

	b, err := FileExists(filename)
	if Error(err) {
		return err
	}

	if !b {
		return &ErrFileNotFound{FileName: filename}
	}

	ba, err := ioutil.ReadFile(filename)
	if Error(err) {
		return err
	}

	gomod := gomodInfo{}
	gomod.Require = make([]require, 0)

	scanner := bufio.NewScanner(bytes.NewReader(ba))
	inRequire := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if inRequire {
			inRequire = line != ")"

			if inRequire {
				req := require{}

				p := strings.Index(line, " ")
				if p != -1 {
					req.Name = strings.TrimSpace(line[:p])
					line = line[p:]
				}

				p = strings.Index(line, "//")
				if p != -1 {
					req.Version = strings.TrimSpace(line[:p])
					line = line[p:]

					req.Comment = strings.TrimSpace(line)
				} else {
					req.Version = strings.TrimSpace(line)
				}

				paths := []string{gopath, "pkg", "mod"}
				paths = append(paths, filepath.SplitList(req.Name)...)
				paths[len(paths)-1] = filepath.Join(fmt.Sprintf("%s@%s", paths[len(paths)-1], req.Version), "LICENSE")

				req.LicenseName = "custom"
				req.LicenseUrl = "not available"
				req.LicenseText = "not available"

				licenseFile := filepath.Join(paths...)

				b, err := FileExists(licenseFile)
				if Error(err) {
					return err
				}

				if b {
					ba, err := ioutil.ReadFile(licenseFile)
					if Error(err) {
						return err
					}

					req.LicenseText = string(ba)
				}

				url := req.Name
				if strings.HasPrefix(url, "github.com/") {
					url = url[11:]
				}

				splits := strings.Split(url, "/") // remove trailing .../v4

				if len(splits) > 1 {
					url = splits[0] + "/" + splits[1]
				}

				ba, err := URLGet("https://api.github.com/repos/" + url)
				if Error(err) {
					return err
				}

				j, err := NewJason(string(ba))
				if Error(err) {
					return err
				}

				e, _ := j.Element("license")
				if e != nil {
					url, err = e.String("url")
					if Error(err) {
						return err
					}

					if url != "" {
						req.LicenseUrl = url

						ba, err := URLGet(req.LicenseUrl)
						if Error(err) {
							return err
						}

						j, err := NewJason(string(ba))
						if Error(err) {
							return err
						}

						body, err := j.String("body")
						if Error(err) {
							return err
						}

						req.LicenseText = body
					}

					name, _ := e.String("name")
					if name != "" && strings.ToLower(name) != "other" {
						req.LicenseName = name
					}
				}

				gomod.Require = append(gomod.Require, req)

				continue
			} else {
				break
			}
		}

		if strings.HasPrefix(line, "module") {
			gomod.Module = strings.TrimSpace(line[len("module"):])

			continue
		}

		if strings.HasPrefix(line, "go ") {
			gomod.Version = strings.TrimSpace(line[len("go "):])

			continue
		}

		if strings.HasPrefix(line, "require") {
			inRequire = true

			continue
		}
	}

	ba, err = json.MarshalIndent(gomod, "", "    ")
	if Error(err) {
		return err
	}

	return ioutil.WriteFile(filepath.Join(path, AppFilename("-opensource.json")), ba, DefaultFileMode)
}
