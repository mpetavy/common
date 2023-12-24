package common

import (
	"fmt"
	"golang.org/x/mod/modfile"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ModuleRequire struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Comment     string `json:"comment"`
	LicenseName string `json:"licenseName"`
	LicenseUrl  string `json:"licenseUrl"`
	LicenseText string `json:"licenseText"`
}

type ModuleInfo struct {
	Disclosure string          `json:"disclosure"`
	Software   string          `json:"software"`
	Version    string          `json:"version"`
	Requires   []ModuleRequire `json:"requires"`
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
	ba, _, err := ReadResource(filename)
	if Error(err) {
		return nil, err
	}

	mf, err := modfile.Parse(filename, ba, nil)

	moduleInfo := ModuleInfo{}
	moduleInfo.Software = mf.Module.Mod.String()
	moduleInfo.Version = Version(true, true, true)

	for _, require := range mf.Require {
		req := ModuleRequire{}
		req.Name = require.Mod.Path
		req.Version = require.Mod.Version

		if require.Indirect {
			req.Comment = "// indirect"
		}

		paths := []string{gopath, "pkg", "mod"}
		paths = append(paths, Split(req.Name, "/")...)
		paths[len(paths)-1] = filepath.Join(fmt.Sprintf("%s@%s", paths[len(paths)-1], req.Version), "LICENSE")

		req.LicenseName = "Custom"
		req.LicenseUrl = ""
		req.LicenseText = ""

		if strings.Index(req.Comment, "indirect") != -1 {
			continue
		}

		licenseFile := filepath.Join(paths...)

		if FileExists(licenseFile) {
			ba, err := os.ReadFile(licenseFile)
			if Error(err) {
				return nil, err
			}

			req.LicenseText = string(ba)
		}

		if strings.Index(req.LicenseText, "Copyright (c) 2009 The Go Authors. All rights reserved.") != -1 {
			req.LicenseName = "Google GO License"
			req.LicenseUrl = "https://golang.org/LICENSE?m=text"
		} else {
			if strings.HasPrefix(req.Name, "github.com/") {
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
		}

		moduleInfo.Requires = append(moduleInfo.Requires, req)
	}

	sort.SliceStable(moduleInfo.Requires, func(i, j int) bool {
		return strings.ToUpper(moduleInfo.Requires[i].Name) < strings.ToUpper(moduleInfo.Requires[j].Name)
	})

	return &moduleInfo, nil
}
