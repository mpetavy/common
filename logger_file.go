package common

import (
	"io"
	"os"
	"path/filepath"
	"sync"
)

type fileWriter struct {
	io.Writer

	mu       sync.Mutex
	file     *os.File
	filesize int
}

func newFileWriter() (*fileWriter, error) {
	fw := &fileWriter{}

	if FileExists(*FlagLogFileName) {
		fi, err := os.Stat(*FlagLogFileName)
		if err != nil {
			return nil, err
		}

		fw.filesize = int(fi.Size())

		fw.file, err = os.OpenFile(*FlagLogFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, DefaultFileMode)
		if err != nil {
			return nil, err
		}
	} else {
		err := fw.createFile()
		if err != nil {
			return nil, err
		}
	}

	return fw, nil
}

func (fw *fileWriter) Write(msg []byte) (int, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.filesize+len(msg) > *FlagLogFileSize {
		err := fw.createFile()
		if err != nil {
			return 0, err
		}
	}

	n, err := fw.file.Write(msg)
	if err != nil {
		return n, err
	}

	fw.filesize += n

	err = fw.file.Sync()
	if err != nil {
		return n, err
	}

	return n, nil
}

func (fw *fileWriter) createFile() error {
	err := fw.closeFile()
	if err != nil {
		return err
	}

	err = FileBackup(*FlagLogFileName)
	if err != nil {
		return err
	}

	dir := filepath.Dir(*FlagLogFileName)
	if !FileExists(dir) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	fw.file, err = os.OpenFile(*FlagLogFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, DefaultFileMode)
	if err != nil {
		return err
	}

	return nil
}

func (fw *fileWriter) closeFile() error {
	if fw.file == nil {
		return nil
	}

	defer func() {
		fw.file = nil
		fw.filesize = 0
	}()

	err := fw.file.Close()
	if err != nil {
		return err
	}

	return nil
}
