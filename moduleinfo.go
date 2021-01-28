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
	pat := os.Getenv("GITHUB_PAT")
	if pat == "" {
		return nil, fmt.Errorf("undefined GITHUB_PAT")
	}

	filename := "go.mod"

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

				if strings.Index(req.Comment, "indirect") != -1 {
					continue
				}

				licenseFile := filepath.Join(paths...)

				if FileExists(licenseFile) {
					ba, err := ioutil.ReadFile(licenseFile)
					if Error(err) {
						return nil, err
					}

					req.LicenseText = string(ba)
				}

				if strings.Index(req.LicenseText, "Copyright (c) 2009 The Go Authors. All rights reserved.") != -1 {
					req.LicenseName = "Google GO License"
					req.LicenseUrl = "https://golang.org/LICENSE?m=text"
				} else {
					url := req.Name
					if strings.HasPrefix(url, "github.com/") {
						url = url[11:]
					}

					splits := strings.Split(url, "/") // remove trailing .../v4

					if len(splits) > 1 {
						url = splits[0] + "/" + splits[1]
					}

					ba, err := URLGet(fmt.Sprintf("https://%s:%s@api.github.com/repos/%s", username, pat, url))
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

							i := strings.Index(url, "//")
							if i != -1 {
								url = fmt.Sprintf("%s%s:%s@%s", url[:i+2], username, pat, url[i+2:])
							}

							ba, err := URLGet(url)
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
			moduleInfo.Version = TitleVersion(true, true, true)

			continue
		}

		if strings.HasPrefix(line, "require") {
			inRequire = true

			continue
		}
	}

	return &moduleInfo, nil
}
