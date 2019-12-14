package common

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
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

type ModuleInfo struct {
	Module  string
	Version string
	Require []require
}

func CreateModuleInfo() (*ModuleInfo, error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return nil, fmt.Errorf("undefined GOPATH")
	}
	username := os.Getenv("GITHUB_USERNAME")
	if username == "" {
		return nil, fmt.Errorf("undefined GITHUB_USERNAME")
	}
	password := os.Getenv("GITHUB_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("undefined GITHUB_PASSWORD")
	}

	filename := "go.mod"

	b, err := FileExists(filename)
	if Error(err) {
		return nil, err
	}

	if !b {
		return nil, &ErrFileNotFound{FileName: filename}
	}

	ba, err := ioutil.ReadFile(filename)
	if Error(err) {
		return nil, err
	}

	moduleInfo := ModuleInfo{}
	moduleInfo.Require = make([]require, 0)

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

				req.LicenseName = "Custom"
				req.LicenseUrl = ""
				req.LicenseText = ""

				licenseFile := filepath.Join(paths...)

				b, err := FileExists(licenseFile)
				if Error(err) {
					return nil, err
				}

				if b {
					ba, err := ioutil.ReadFile(licenseFile)
					if Error(err) {
						return nil, err
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

				ba, err := URLGet(fmt.Sprintf("https://%s:%s@api.github.com/repos/%s", username, password, url))
				if Error(err) {
					return nil, err
				}

				j, err := NewJason(string(ba))
				if Error(err) {
					return nil, err
				}

				e, _ := j.Element("license")
				if e != nil {
					url, err = e.String("url")
					if Error(err) {
						return nil, err
					}

					if url != "" {
						req.LicenseUrl = url

						ba, err := URLGet(req.LicenseUrl)
						if Error(err) {
							return nil, err
						}

						j, err := NewJason(string(ba))
						if Error(err) {
							return nil, err
						}

						body, err := j.String("body")
						if Error(err) {
							return nil, err
						}

						req.LicenseText = body

						url, err = j.String("html_url")
						if Error(err) {
							return nil, err
						}

						if url != "" {
							req.LicenseUrl = url
						}
					}

					name, _ := e.String("name")
					if name != "" && strings.ToLower(name) != "other" {
						req.LicenseName = name
					}
				}

				moduleInfo.Require = append(moduleInfo.Require, req)

				continue
			} else {
				break
			}
		}

		if strings.HasPrefix(line, "module") {
			moduleInfo.Module = strings.TrimSpace(line[len("module"):])

			continue
		}

		if strings.HasPrefix(line, "go ") {
			moduleInfo.Version = strings.TrimSpace(line[len("go "):])

			continue
		}

		if strings.HasPrefix(line, "require") {
			inRequire = true

			continue
		}
	}

	return &moduleInfo, nil
}
