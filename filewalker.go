package common

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Filewalker struct {
	Path        string
	Filemask    string
	Recursive   bool
	IgnoreError bool
	fileFunc    func(path string, f os.FileInfo) error
}

func (fw *Filewalker) walkfunc(path string, f os.FileInfo, err error) error {
	if err != nil {
		if fw.IgnoreError {
			Warn(fmt.Errorf("cannot access: %s", path))

			return filepath.SkipDir
		}

		return err
	}

	if f.IsDir() {
		if path == fw.Path || fw.Recursive {
			return fw.fileFunc(path, f)
		} else {
			return fs.SkipDir
		}
	} else {
		b := fw.Filemask == ""

		if !b {
			b, err = EqualWildcards(filepath.Base(path), fw.Filemask)
			if Error(err) {
				return err
			}
		}

		if !b {
			return nil
		}

		return fw.fileFunc(path, f)
	}
}

func (fw *Filewalker) Run() error {
	if !FileExists(fw.Path) || !IsDirectory(fw.Path) {
		return &ErrFileNotFound{
			FileName: fw.Path,
		}
	}

	return filepath.Walk(fw.Path, fw.walkfunc)
}

func NewFilewalker(filemask string, recursive bool, ignoreError bool, walkFunc func(path string, f os.FileInfo) error) (*Filewalker, error) {
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
			return nil, &ErrFileNotFound{
				FileName: filemask,
			}
		}
	}

	return &Filewalker{
		Path:        path,
		Filemask:    filemask,
		Recursive:   recursive,
		IgnoreError: ignoreError,
		fileFunc:    walkFunc,
	}, nil
}
