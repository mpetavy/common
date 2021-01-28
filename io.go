package common

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	ReadOnlyFileMode = FileMode(true, true, false)
	DefaultFileMode  = FileMode(true, true, false)
	DefaultDirMode   = FileMode(true, true, true)
	FlagCountBackups *int
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

	FlagCountBackups = flag.Int("bak", 3, "amount of file backups")
}

// AppCleanup cleans up all remaining objects
func deleteTempDir() error {
	if !FileExists(tempDir) {
		return nil
	}

	DebugFunc(tempDir)

	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			b, err := IsFileReadOnly(path)
			if Error(err) {
				return err
			}

			if !b {
				err := SetFileReadOnly(path, false)
				if Error(err) {
					return err
				}
			}
		}

		return nil
	})
	if Error(err) {
		return err
	}

	err = os.RemoveAll(tempDir)
	if Error(err) {
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
	if Error(err) {
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
	if Error(err) {
		return "", err
	}

	Debug(fmt.Sprintf("CreateTempDir : %s", tempdir))

	return tempdir, err
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)

	if os.IsNotExist(err) || err != nil {
		return false
	} else {
		return true
	}
}

// FileExists does ... guess what :-)
func FileExists(filename string) bool {
	b := fileExists(filename)

	Debug(fmt.Sprintf("FileExists %s: %v", filename, b))

	return b
}

// FileDelete does ... guess what :-)
func FileDelete(filename string) error {
	if FileExists(filename) {
		Debug(fmt.Sprintf("FileRemove %s", filename))

		err := os.Remove(filename)
		if Error(err) {
			return err
		}
	}

	return nil
}

// FileDate does ... guess what :-)
func FileDate(filename string) (time.Time, error) {
	f, err := os.Stat(filename)
	if Error(err) {
		return time.Time{}, err
	}

	t, err := f.ModTime().MarshalText()
	if Error(err) {
		return time.Time{}, err
	}

	Debug(fmt.Sprintf("FileDate %s: %s", filename, string(t)))

	return f.ModTime(), nil
}

// FileSize does ... guess what :-)
func FileSize(filename string) (int64, error) {
	file, err := os.Stat(filename)
	if Error(err) {
		return -1, err
	}

	return file.Size(), nil
}

// FileCopy does ... guess what :-)
func FileCopy(src string, dst string) error {
	srcFile, err := os.Open(src)
	if Error(err) {
		return err
	}
	defer func() {
		Error(srcFile.Close())
	}()

	destFile, err := os.Create(dst)
	if Error(err) {
		return err
	}
	defer func() {
		Error(destFile.Close())
		Error(destFile.Sync())
	}()

	_, err = io.Copy(destFile, srcFile)
	if Error(err) {
		return err
	}

	return nil
}

// FileStore creates backup of files
func FileStore(filename string, r io.Reader) error {
	// create the file
	out, err := os.Create(filename)
	if Error(err) {
		return err
	}

	// care about final cleanup of open file
	defer func() {
		Ignore(out.Close())
	}()

	// download the remote resource to the file
	_, err = io.Copy(out, r)
	if Error(err) {
		return err
	}

	return nil
}

// FileBackup creates backup of files
func FileBackup(filename string) error {
	if *FlagCountBackups < 1 {
		return nil
	}

	for i := *FlagCountBackups - 1; i >= 0; i-- {
		src := filename
		if i > 0 {
			src = src + "." + strconv.Itoa(i)
		}

		dst := ""
		if *FlagCountBackups == 1 {
			dst = filename + ".bak"
		} else {
			dst = filename + "." + strconv.Itoa(i+1)
		}

		if fileExists(src) {
			if fileExists(dst) {
				err := FileDelete(dst)
				if Error(err) {
					return err
				}
			}

			err := os.Rename(src, dst)
			if Error(err) {
				return err
			}
		}
	}

	return nil
}

// IsFileReadOnly checks if a file is read only
func IsFileReadOnly(path string) (result bool, err error) {
	result = false

	file, err := os.OpenFile(path, os.O_WRONLY, DefaultFileMode)
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
	if FileExists(path) {
		fi, err := os.Stat(path)
		if Error(err) {
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
	if Error(err) {
		return false, err
	}

	return !b, nil
}

// IsSymbolicLink checks if the path leads to symbolic link
func IsSymbolicLink(path string) bool {
	file, err := os.Lstat(path)
	if Error(err) {
		return false
	}

	return file.Mode()&os.ModeSymlink != 0
}

// SetFileReadOnly sets file READ-ONLY yes or false
func SetFileReadOnly(path string, readonly bool) (err error) {
	if readonly {
		err = os.Chmod(path, ReadOnlyFileMode)
	} else {
		err = os.Chmod(path, DefaultFileMode)
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

		if IsRunningInteractive() {
			dir, err = os.Getwd()
		} else {
			dir, err = os.Executable()
			if err == nil {
				dir = filepath.Dir(dir)
			}
		}

		if !Error(err) {
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

func CopyWithContext(ctx context.Context, cancel context.CancelFunc, name string, writer io.Writer, reader io.Reader, bufferSize int) (int64, error) {
	Debug("%s copyWithContext: start", name)

	var written int64
	var err error

	go func() {

		if bufferSize <= 0 {
			bufferSize = 32 * 1024
		}

		buf := make([]byte, bufferSize)

		defer func() {
			Debug("%s cancel!", name)
			cancel()
		}()

		if *FlagLogIO {
			written, err = CopyBufferError(io.CopyBuffer(io.MultiWriter(writer, &debugWriter{name, "WRITE"}), io.TeeReader(reader, &debugWriter{name, "READ"}), buf))
		} else {
			written, err = CopyBufferError(io.CopyBuffer(writer, reader, buf))
		}
	}()

	<-ctx.Done()

	Debug("%s copyWithContext: stop", name)

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

type lineBuffer struct {
	buf   bytes.Buffer
	count int
	lines []string
	f     func(string) string
	ba    io.Reader
}

func NewLineBuffer(count int, f func(string) string) *lineBuffer {
	return &lineBuffer{
		buf:   bytes.Buffer{},
		count: count,
		lines: make([]string, 0),
		f:     f,
	}
}

func (this *lineBuffer) Read(p []byte) (n int, err error) {
	if this.ba == nil {
		this.ba = strings.NewReader(strings.Join(this.lines, ""))
	}

	return this.ba.Read(p)
}

func (this *lineBuffer) Write(p []byte) (int, error) {
	for _, b := range p {
		err := this.buf.WriteByte(b)
		if Error(err) {
			return -1, err
		}

		if b == '\n' {
			line := this.buf.String()
			if this.f != nil {
				line = this.f(line)
			}

			if len(this.lines) < this.count {
				this.lines = append(this.lines, line)
			} else {
				copy(this.lines, this.lines[1:])
				this.lines[len(this.lines)-1] = line
			}
			this.buf.Reset()
		}
	}

	return len(p), nil
}

func (this *lineBuffer) Lines() []string {
	return this.lines
}

func URLGet(url string) ([]byte, error) {
	h := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if Error(err) {
		return nil, err
	}

	var r *http.Response

	r, err = h.Do(req)
	if Error(err) {
		return nil, err
	}

	ba, err := ioutil.ReadAll(r.Body)

	defer func() {
		Error(r.Body.Close())
	}()

	if Error(err) {
		return nil, err
	}

	return ba, nil
}

func WriteJsonFile(filename string, v interface{}, fileMode os.FileMode) error {
	ba, err := json.MarshalIndent(v, "", "    ")
	if Error(err) {
		return err
	}

	return ioutil.WriteFile(filename, ba, fileMode)
}

func ReadJsonFile(filename string, v interface{}) error {
	ba, err := ioutil.ReadFile(filename)
	if Error(err) {
		return err
	}

	return json.Unmarshal(ba, v)
}

type ZeroReader struct {
}

func NewZeroReader() *ZeroReader {
	return &ZeroReader{}
}

func (this ZeroReader) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = 0
	}

	return len(p), nil
}

type RandomReader struct {
	template [256]byte
}

func NewRandomReader() *RandomReader {
	r := RandomReader{}

	for i := range r.template {
		r.template[i] = byte(Rnd(256))
	}

	return &r
}

func (this RandomReader) Read(p []byte) (n int, err error) {
	copy(p, this.template[:])

	return len(p), nil
}

type DeadlineReader struct {
	reader  io.Reader
	timeout time.Duration
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewDeadlineReader(reader io.Reader, timeout time.Duration) io.Reader {
	return &DeadlineReader{
		reader:  reader,
		timeout: timeout,
	}
}

func (this *DeadlineReader) Read(p []byte) (int, error) {
	if this.ctx == nil {
		this.ctx, this.cancel = context.WithDeadline(context.Background(), time.Now().Add(this.timeout))
	}

	select {
	case <-this.ctx.Done():
		this.cancel()

		return 0, io.EOF
	default:
		return this.reader.Read(p)
	}
}

type TimeoutReader struct {
	reader   io.Reader
	timeout  time.Duration
	useTimer bool

	FirstRead time.Time
}

func NewTimeoutReader(reader io.Reader, timeout time.Duration, initalTimeout bool) *TimeoutReader {
	return &TimeoutReader{
		reader:   reader,
		timeout:  timeout,
		useTimer: initalTimeout,
	}
}

func (this *TimeoutReader) Read(p []byte) (int, error) {
	if !this.useTimer {

		this.useTimer = true

		n, err := this.reader.Read(p)

		this.FirstRead = time.Now()

		return n, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), this.timeout)
	defer cancel()

	var n int
	var err error

	ch := make(chan interface{})

	go func() {
		n, err = this.reader.Read(p)

		close(ch)
	}()

	select {
	case <-ctx.Done():
		return 0, io.EOF
	case <-ch:
		return n, err
	}
}
