package common

import (
	"io/fs"
	"os"
	"path/filepath"
)

type filewalker struct {
	Path        string
	Filemask    string
	Recursive   bool
	IgnoreError bool
	walkFunc    func(path string, f os.FileInfo) error
}

func (fw *filewalker) walkfunc(path string, f os.FileInfo, err error) error {
	if err != nil {
		if fw.IgnoreError {
			Warn("cannot access: %s", path)

			return filepath.SkipDir
		}

		return err
	}

	if f.IsDir() {
		if path == fw.Path || fw.Recursive {
			return fw.walkFunc(path, f)
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

		return fw.walkFunc(path, f)
	}
}

func WalkFiles(filemask string, recursive bool, ignoreError bool, walkFunc func(path string, fi os.FileInfo) error) error {
	path := ""
	filemask = CleanPath(filemask)

	if ContainsWildcard(filemask) || !FileExists(filemask) {
		path = filepath.Dir(filemask)
		filemask = filepath.Base(filemask)
	} else {
		if IsDirectory(filemask) {
			path = filemask
			filemask = ""
		} else {
			path = filepath.Dir(filemask)
			filemask = filepath.Base(filemask)
		}
	}

	if !FileExists(path) || !IsDirectory(path) {
		return &ErrFileNotFound{
			FileName: path,
		}
	}

	fw := &filewalker{
		Path:        path,
		Filemask:    filemask,
		Recursive:   recursive,
		IgnoreError: ignoreError,
		walkFunc:    walkFunc,
	}

	return filepath.Walk(fw.Path, fw.walkfunc)
}

func ListFiles(filemask string, recursive bool) ([]string, error) {
	var files []string

	err := WalkFiles(filemask, recursive, false, func(path string, fi os.FileInfo) error {
		if fi.IsDir() {
			return nil
		}

		files = append(files, path)

		return nil
	})
	if Error(err) {
		return nil, err
	}

	return files, nil
}
