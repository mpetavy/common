package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Filewalker struct {
	Path                    string
	Filemask                string
	Recursive               bool
	IgnoreError             bool
	IgnoreHiddenDirectories bool
	walkFunc                func(path string, f os.FileInfo) error
}

func (this *Filewalker) Walkfunc(path string, f os.FileInfo, err error) error {
	if err != nil {
		if this.IgnoreError {
			Warn(fmt.Errorf("cannot access: %s", path))

			return filepath.SkipDir
		}

		return err
	}

	if !f.IsDir() {
		b := this.Filemask == ""

		if !b {
			b, err = EqualWildcards(filepath.Base(path), this.Filemask)
			if Error(err) {
				return err
			}
		}

		if !b {
			return nil
		}

		return this.walkFunc(path, f)
	}

	if this.IgnoreHiddenDirectories && strings.HasPrefix(f.Name(), ".") {
		return filepath.SkipDir
	}

	if this.Recursive || path == this.Path {
		return nil
	}

	return filepath.SkipDir
}

func (this *Filewalker) Run() error {
	return filepath.Walk(this.Path, this.Walkfunc)
}

func NewFilewalker(filemask string, recursive bool, ignoreError bool, walkFunc func(path string, f os.FileInfo) error) *Filewalker {
	path := ""
	filemask = CleanPath(filemask)

	if ContainsWildcard(filemask) {
		path = filepath.Dir(filemask)
		filemask = filepath.Base(filemask)
	} else {
		if FileExists(filemask) {
			if IsDirectory(filemask) {
				path = filemask
				filemask = ""
			} else {
				path = filepath.Dir(filemask)
				filemask = filepath.Base(filemask)
			}
		} else {
			path = filepath.Dir(filemask)
			filemask = filepath.Base(filemask)
		}
	}

	return &Filewalker{
		Path:        path,
		Filemask:    filemask,
		Recursive:   recursive,
		IgnoreError: ignoreError,
		walkFunc:    walkFunc,
	}
}
