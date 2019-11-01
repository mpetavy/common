package common

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/kardianos/service"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	FileFileMode = FileMode(true, true, false)
	DirFileMode  = FileMode(true, true, true)
	countBackups *int
)

type ErrFileNotFound struct {
	FileName string
}

func (e *ErrFileNotFound) Error() string {
	return fmt.Sprintf("file or path not found: %s", e.FileName)
}

type debugWriter struct {
	Name   string
	Action string
}

func (this *debugWriter) Write(p []byte) (n int, err error) {
	Debug("%s %s %d bytes: %+q", this.Name, this.Action, len(p), string(p))

	return len(p), nil
}

// +-----+---+--------------------------+
// | rwx | 7 | Read, write and execute  |
// | rw- | 6 | Read, write              |
// | r-x | 5 | Read, and execute        |
// | r-- | 4 | Read,                    |
// | -wx | 3 | Write and execute        |
// | -w- | 2 | Write                    |
// | --x | 1 | Execute                  |
// | --- | 0 | no permissions           |
// +------------------------------------+

// +------------+------+-------+
// | Permission | Octal| Field |
// +------------+------+-------+
// | rwx------  | 0700 | User  |
// | ---rwx---  | 0070 | Group |
// | ------rwx  | 0007 | Other |
// +------------+------+-------+

var tempDir string

func init() {
	var err error

	tempDir, err = ioutil.TempDir("", Title())
	if err != nil {
		panic(err)
	}

	AddShutdownHook(func() {
		Error(deleteTempDir())
	})

	countBackups = flag.Int("filebackup", 1, "amount of file backups")
}

// AppCleanup cleans up all remaining objects
func deleteTempDir() error {
	b, err := FileExists(tempDir)
	if err != nil {
		return err
	}

	if !b {
		return nil
	}

	DebugFunc(tempDir)

	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			b, err := IsFileReadOnly(path)
			if err != nil {
				return err
			}

			if !b {
				err := SetFileReadOnly(path, false)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = os.RemoveAll(tempDir)
	if err != nil {
		return err
	}

	return nil
}

// TempDir returns the private temporary directory of the app
func TempDir() string {
	return tempDir
}

// CreateTempFile creates a temporary file
func CreateTempFile() (file *os.File, err error) {
	tempDir := TempDir()

	file, err = ioutil.TempFile(tempDir, GetRuntimeInfo(1).Filename()+"-")
	if err != nil {
		return nil, err
	}
	defer func() {
		Ignore(file.Close())
	}()

	Debug(fmt.Sprintf("CreateTempFile : %s", file.Name()))

	return file, err
}

// CreateTempDir creates a temporary file
func CreateTempDir() (string, error) {
	rootTempDir := TempDir()

	tempdir, err := ioutil.TempDir(rootTempDir, GetRuntimeInfo(1).Filename()+"-")
	if err != nil {
		return "", err
	}

	Debug(fmt.Sprintf("CreateTempDir : %s", tempdir))

	return tempdir, err
}

func fileExists(filename string) (bool, error) {
	var b bool
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		b = false
		err = nil
	} else {
		b = err == nil
	}

	return b, err
}

// FileExists does ... guess what :-)
func FileExists(filename string) (bool, error) {
	b, err := fileExists(filename)

	Debug(fmt.Sprintf("FileExists %s: %v", filename, b))

	return b, err
}

// FileDelete does ... guess what :-)
func FileDelete(filename string) error {
	b, err := FileExists(filename)
	if err != nil {
		return err
	}

	if b {
		Debug(fmt.Sprintf("FileRemove %s: %v", filename, b))

		return os.Remove(filename)
	}

	return nil
}

// FileDate does ... guess what :-)
func FileDate(filename string) (time.Time, error) {
	f, err := os.Stat(filename)
	if err != nil {
		return time.Time{}, err
	}

	t, err := f.ModTime().MarshalText()
	if err != nil {
		return time.Time{}, err
	}

	Debug(fmt.Sprintf("FileDate %s: %s", filename, string(t)))

	return f.ModTime(), nil
}

// FileSize does ... guess what :-)
func FileSize(filename string) (int64, error) {
	file, err := os.Stat(filename)
	if err != nil {
		return -1, err
	}

	return file.Size(), nil
}

// FileCopy does ... guess what :-)
func FileCopy(src string, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		Ignore(srcFile.Close())
	}()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		Ignore(destFile.Close())
	}()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	err = destFile.Sync()
	if err != nil {
		return err
	}

	return nil
}

// FileStore creates backup of files
func FileStore(filename string, r io.Reader) error {
	// create the file
	out, err := os.Create(filename)
	if err != nil {
		return err
	}

	// care about final cleanup of open file
	defer func() {
		Ignore(out.Close())
	}()

	// download the remote resource to the file
	_, err = io.Copy(out, r)
	if err != nil {
		return err
	}

	return nil
}

// FileBackup creats backup of files
func FileBackup(filename string) error {
	if *countBackups < 1 {
		return nil
	}

	for i := *countBackups - 1; i >= 0; i-- {
		src := filename
		if i > 0 {
			src = src + "." + strconv.Itoa(i)
		}

		dst := ""
		if *countBackups == 1 {
			dst = filename + ".bak"
		} else {
			dst = filename + "." + strconv.Itoa(i+1)
		}

		b, err := FileExists(src)
		if err != nil {
			continue
		}

		if b {
			err = FileDelete(dst)
			if err != nil {
				continue
			}

			err := os.Rename(src, dst)
			if err != nil {
				continue
			}
		}
	}

	return nil
}

// IsFileReadOnly checks if a file is read only
func IsFileReadOnly(path string) (result bool, err error) {
	result = false

	file, err := os.OpenFile(path, os.O_WRONLY, FileFileMode)
	if err != nil {
		if !os.IsPermission(err) {
			result = true
		} else {
			return false, err
		}
	}
	Ignore(file.Close())

	return result, nil
}

// IsDirectory checks if the path leads to a directory
func IsDirectory(path string) (bool, error) {
	b, err := FileExists(path)
	if err != nil {
		return false, err
	}

	if b {
		fi, err := os.Stat(path)
		if err != nil {
			return false, err
		}

		return fi.IsDir(), nil
	} else {
		return false, nil
	}
}

// IsDirectory checks if the path leads to a directory
func IsFile(path string) (bool, error) {
	b, err := IsDirectory(path)
	if err != nil {
		return false, err
	}

	return !b, nil
}

// IsSymbolicLink checks if the path leads to symbolic link
func IsSymbolicLink(path string) bool {
	file, err := os.Lstat(path)
	if err != nil {
		return false
	}

	return file.Mode()&os.ModeSymlink != 0
}

// SetFileReadOnly sets file READ-ONLY yes or false
func SetFileReadOnly(path string, readonly bool) (err error) {
	if readonly {
		err = os.Chmod(path, FileMode(true, false, false))
	} else {
		err = os.Chmod(path, FileFileMode)
	}

	return err
}

// Returns the complete filename "test.txt"
func FileName(filename string) string {
	_, filename = filepath.Split(filename)

	return filename
}

// Returns the filename part without extension "test.txt" -> "test"
func FileNamePart(filename string) string {
	_, filename = filepath.Split(filename)

	return filename[0 : len(filename)-len(FileNameExt(filename))]
}

// Returns the filename extension without part "test.txt" -> ".txt"
func FileNameExt(filename string) string {
	return filepath.Ext(filename)
}

// CleanPath cleans the given path and also replace to OS specific separators
func CleanPath(path string) string {
	result := path

	if IsWindowsOS() {
		result = strings.Replace(result, "/", string(filepath.Separator), -1)
	} else {
		result = strings.Replace(result, "\\", string(filepath.Separator), -1)
	}

	p := strings.Index(result, "~")

	if p != -1 {
		userHomeDir := ""

		usr, err := user.Current()
		if !Error(err) {
			userHomeDir = usr.HomeDir
		}

		result = strings.Replace(result, "~", userHomeDir, -1)
	}

	result = filepath.Clean(result)

	if !filepath.IsAbs(result) && !strings.HasPrefix(result, string(filepath.Separator)) {
		var dir string
		var err error

		if service.Interactive() {
			dir, err = os.Getwd()
		} else {
			dir, err = os.Executable()
			if err == nil {
				dir = filepath.Dir(dir)
			}
		}

		if err != nil {
			Error(err)
		} else {
			result = filepath.Join(dir, result)
		}
	}

	DebugFunc("%s -> %s", path, result)

	return result
}

func ScanLinesWithLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, data[0 : i+1], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func CopyWithContext(ctx context.Context, cancel context.CancelFunc, name string, writer io.Writer, reader io.Reader) (int64, error) {
	Debug("%s copyWithContext: start", name)

	var written int64
	var err error

	go func(written *int64, err error) {
		defer func() {
			Debug("%s cancel!", name)
			cancel()
		}()
		*written, err = io.Copy(io.MultiWriter(writer, &debugWriter{name, "WRITE"}), io.TeeReader(reader, &debugWriter{name, "READ"}))
		if err != nil {
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				err = fmt.Errorf("Timeoutout error")
			}
			WarnError(err)
		}
	}(&written, err)

	select {
	case <-ctx.Done():
		Debug("%s copyWithContext: stop", name)
	}

	return written, err
}

type FilePermission struct {
	Read    bool
	Write   bool
	Execute bool
}

func CalcFileMode(owner FilePermission, group FilePermission, public FilePermission) os.FileMode {
	txt := "0"

	for _, p := range []FilePermission{owner, group, public} {
		var value int

		if p.Execute {
			value += 1
		}
		if p.Write {
			value += 2
		}
		if p.Read {
			value += 4
		}

		txt += strconv.Itoa(value)
	}

	result, _ := strconv.ParseInt(txt, 8, 64)

	DebugFunc("%s, %d: owner: %+v group: %+v public: %+v", txt, result, owner, group, public)

	return os.FileMode(result)
}

func FileMode(read, write, execute bool) os.FileMode {
	return CalcFileMode(
		FilePermission{
			Read:    read,
			Write:   write,
			Execute: execute,
		},
		FilePermission{
			Read:    read,
			Write:   false,
			Execute: execute,
		},
		FilePermission{
			Read:    read,
			Write:   false,
			Execute: execute,
		},
	)
}
