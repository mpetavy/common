package common

import (
	"os"
	"path/filepath"
)

type filewalker struct {
	path      string
	filemask  string
	recursive bool
	walkFunc  func(path string) error
}

func (this *filewalker) walkfunc(path string, fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	b, err := IsFile(path)
	if err != nil {
		return err
	}

	if b {
		b = this.filemask == ""

		if !b {
			b, err = EqualWildcards(filepath.Base(path), this.filemask)
			if err != nil {
				return err
			}
		}

		if !b {
			return nil
		}

		return this.walkFunc(path)
	}

	if this.recursive || path == this.path {
		return nil
	}

	return filepath.SkipDir
}

func WalkFilepath(filemask string, recursive bool, walkFunc func(path string) error) error {
	path := ""
	filemask = CleanPath(filemask)

	if ContainsWildcard(filemask) {
		path = filepath.Dir(filemask)
		filemask = filepath.Base(filemask)
	} else {
		b, err := FileExists(filemask)
		if err != nil {
			return err
		}

		if b {
			b, err := IsDirectory(filemask)
			if err != nil {
				return err
			}

			if b {
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

	w := filewalker{path: path, filemask: filemask, recursive: recursive, walkFunc: walkFunc}

	return filepath.Walk(path, w.walkfunc)
}