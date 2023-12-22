package common

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

type Module struct {
	Name string
}

type GO struct {
	Version string
}

type Toolchain struct {
	Version string
}

type Require struct {
	Module   string
	Version  string
	Indirect bool
}

type Replace struct {
	Module string
	Target string
}

type Exclude struct {
	Module  string
	Version string
}

type Retract struct {
	VersionLow  string
	VersionHigh string
	Comment     string
}

type Modfile struct {
	Module    Module
	GO        GO
	Toolchain Toolchain
	Requires  []Require
	Replaces  []Replace
	Excludes  []Exclude
	Retracts  []Retract
}

func ReadModfile(ba []byte) (*Modfile, error) {
	modfile := &Modfile{}

	correct := false
	inRequire := false
	inReplace := false
	inExclude := false
	inRetract := false

	var line string

	readRequire := func() {
		if strings.Contains(line, ")") {
			inRequire = false

			return
		}

		indirect := false

		p := strings.Index(line, "// indirect")
		if p != -1 {
			line = strings.TrimSpace(line[:p])

			indirect = true
		}

		splits := Split(line, " ")

		modfile.Requires = append(modfile.Requires, Require{
			Module:   splits[0],
			Version:  splits[1],
			Indirect: indirect,
		})
	}

	readReplace := func() {
		if strings.Contains(line, ")") {
			inReplace = false

			return
		}

		splits := Split(line, " ")

		modfile.Replaces = append(modfile.Replaces, Replace{
			Module: splits[1],
			Target: splits[3],
		})
	}

	readExclude := func() {
		if strings.Contains(line, ")") {
			inExclude = false

			return
		}

		splits := Split(line, " ")

		modfile.Excludes = append(modfile.Excludes, Exclude{
			Module:  splits[1],
			Version: splits[2],
		})
	}

	readRetract := func() {
		if strings.Contains(line, ")") {
			inRetract = false

			return
		}

		comment := ""

		p := strings.Index(line, "//")
		if p != -1 {
			comment = strings.TrimSpace(line[p+2:])
			line = strings.TrimSpace(line[:p])
		}

		retract := Retract{
			Comment: comment,
		}

		splits := Split(line, " ")

		if strings.HasPrefix(splits[1], "[") {
			splits[1] = strings.TrimPrefix(splits[1], "[")
			splits[1] = strings.TrimSuffix(splits[1], "]")
		}

		values := Split(splits[1], ",")

		if len(values) >= 1 {
			retract.VersionLow = values[0]
		}
		if len(values) == 2 {
			retract.VersionHigh = values[1]
		}

		modfile.Retracts = append(modfile.Retracts, retract)
	}

	scanner := bufio.NewScanner(bytes.NewReader(ba))
	for scanner.Scan() {
		line = strings.TrimSpace(scanner.Text())

		switch {
		case inRequire:
			readRequire()
		case inReplace:
			readReplace()
		case inExclude:
			readExclude()
		case inRetract:
			readRetract()
		case strings.Contains(line, "require "):
			p := strings.Index(line, "(")
			if p != -1 {
				inRequire = true
			} else {
				line = strings.TrimSpace(line[p+1:])

				readRequire()
			}
		case strings.Contains(line, "replace "):
			p := strings.Index(line, "(")
			if p != -1 {
				inReplace = true
			} else {
				line = strings.TrimSpace(line[p+1:])

				readReplace()
			}
		case strings.Contains(line, "exclude "):
			p := strings.Index(line, "(")
			if p != -1 {
				inExclude = true
			} else {
				line = strings.TrimSpace(line[p+1:])

				readExclude()
			}
		case strings.Contains(line, "retract "):
			p := strings.Index(line, "(")
			if p != -1 {
				inRetract = true
			} else {
				line = strings.TrimSpace(line[p+1:])

				readRetract()
			}
		case strings.Contains(line, "module "):
			correct = true

			splits := Split(line, " ")

			modfile.Module.Name = splits[1]
		case strings.Contains(line, "go "):
			splits := Split(line, " ")

			modfile.GO.Version = splits[1]
		case strings.Contains(line, "toolchain "):
			splits := Split(line, " ")

			modfile.Toolchain.Version = splits[1]
		}
	}

	if !correct {
		return nil, fmt.Errorf("cannot read go.mod content")
	}

	return modfile, nil
}

func (mf *Modfile) Title() string {
	title := mf.Module.Name
	p := strings.LastIndex(title, "/")

	if p != -1 {
		title = title[p+1:]
	}

	return title
}
