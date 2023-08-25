package common

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	ctxio "github.com/jbenet/go-context/io"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	FlagNameIoFileBackups = "io.file.backups"
)

var (
	ReadOnlyFileMode = FileMode(true, false, false)
	DefaultFileMode  = FileMode(true, true, false)
	DefaultDirMode   = FileMode(true, true, true)

	FlagIoFileBackups *int
)

type ErrFileNotFound struct {
	FileName string
}

func (e *ErrFileNotFound) Error() string {
	return fmt.Sprintf("file or path not found: %s", e.FileName)
}

type ErrFileIsEmpty struct {
	FileName string
}

func (e *ErrFileIsEmpty) Error() string {
	return fmt.Sprintf("file is empty: %s", e.FileName)
}

type ErrFileAlreadyExists struct {
	FileName string
}

func (e *ErrFileAlreadyExists) Error() string {
	return fmt.Sprintf("file or path already exists: %s", e.FileName)
}

type ErrSetTimeout struct {
	Mode string
}

func (e *ErrSetTimeout) Error() string {
	return fmt.Sprintf("cannot set %s timeout", e.Mode)
}

type debugWriter struct {
	Name   string
	Action string
}

func (this *debugWriter) Write(p []byte) (n int, err error) {
	Debug("%s %s %d bytes: %s", this.Name, this.Action, len(p), PrintBytes(p, false))

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

	tempDir, err = os.MkdirTemp("", Title())
	if Error(err) {
		tempDir = os.TempDir()
	} else {
		Events.AddListener(EventShutdown{}, func(event Event) {
			Error(deleteTempDir())
		})
	}

	FlagIoFileBackups = flag.Int(FlagNameIoFileBackups, 3, "amount of file backups")
}

// AppCleanup cleans up all remaining objects
func deleteTempDir() error {
	if !FileExists(tempDir) {
		return nil
	}

	DebugFunc(tempDir)

	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && !IsFileReadOnly(path) {
			err := SetFileReadOnly(path, false)
			if Error(err) {
				return err
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

	file, err = os.CreateTemp(tempDir, GetRuntimeInfo(1).Filename()+"-")
	if Error(err) {
		return nil, err
	}
	defer func() {
		DebugError(file.Close())
	}()

	Debug(fmt.Sprintf("CreateTempFile : %s", file.Name()))

	return file, err
}

// CreateTempDir creates a temporary file
func CreateTempDir() (string, error) {
	rootTempDir := TempDir()

	tempdir, err := os.MkdirTemp(rootTempDir, GetRuntimeInfo(1).Filename()+"-")
	if Error(err) {
		return "", err
	}

	Debug(fmt.Sprintf("CreateTempDir : %s", tempdir))

	return tempdir, err
}

func FileExists_(filename string) bool {
	_, err := os.Stat(filename)

	if os.IsNotExist(err) || err != nil {
		return false
	} else {
		return true
	}
}

// FileExists does ... guess what :-)
func FileExists(filename string) bool {
	b := FileExists_(filename)

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
		DebugError(out.Close())
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
	if *FlagIoFileBackups < 1 {
		return nil
	}

	for i := *FlagIoFileBackups - 1; i >= 0; i-- {
		src := filename
		if i > 0 {
			src = src + "." + strconv.Itoa(i)
		}

		dst := ""
		if *FlagIoFileBackups == 1 {
			dst = filename + ".bak"
		} else {
			dst = filename + "." + strconv.Itoa(i+1)
		}

		if FileExists_(src) {
			if FileExists_(dst) {
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
func IsFileReadOnly(path string) bool {
	file, err := os.OpenFile(path, os.O_WRONLY, DefaultFileMode)
	if !os.IsPermission(err) {
		return true
	}

	DebugError(file.Close())

	return false
}

// IsDirectory checks if the path leads to a directory
func IsDirectory(path string) bool {
	fi, err := os.Stat(path)

	return err == nil && fi.IsDir()
}

// IsDirectory checks if the path leads to a directory
func IsFile(path string) bool {
	return !IsDirectory(path) && !IsSymbolicLink(path)
}

// IsSymbolicLink checks if the path leads to symbolic link
func IsSymbolicLink(path string) bool {
	file, err := os.Lstat(path)

	return err == nil && file.Mode()&os.ModeSymlink != 0
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

	if IsWindows() {
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

		if IsRunningInteractive() || strings.HasPrefix(path, "."+string(filepath.Separator)) {
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

	r := strings.NewReplacer("\"", "")
	result = r.Replace(result)

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

func CopyBuffer(ctx context.Context, cancel context.CancelFunc, name string, writer io.Writer, reader io.Reader, bufferSize int) (int64, error) {
	Debug("CopyBuffer %s start", name)

	defer func() {
		Debug("CopyBuffer %s stop", name)
		cancel()
	}()

	if bufferSize <= 0 {
		bufferSize = 32 * 1024
	}

	buf := make([]byte, bufferSize)

	if *FlagLogIO {
		writer = io.MultiWriter(writer, &debugWriter{name, "WRITE"})
		reader = io.TeeReader(reader, &debugWriter{name, "READ"})
	}

	return io.CopyBuffer(ctxio.NewWriter(ctx, writer), ctxio.NewReader(ctx, reader), buf)
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
	DebugFunc(url)

	h := &http.Client{}

	r, err := h.Get(url)
	if Error(err) {
		return nil, err
	}

	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(r.Status)
	}

	ba, err := io.ReadAll(r.Body)

	defer func() {
		Error(r.Body.Close())
	}()

	if Error(err) {
		return nil, err
	}

	return ba, nil
}

func WriteJsonFile(filename string, v interface{}, fileMode os.FileMode) error {
	ba, err := json.MarshalIndent(v, "", "  ")
	if Error(err) {
		return err
	}

	return os.WriteFile(filename, ba, fileMode)
}

func ReadJsonFile(filename string, v interface{}) error {
	ba, err := os.ReadFile(filename)
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

type TimeoutReader struct {
	FirstRead      time.Time
	initialTimeout bool
	canSet         bool
	reader         io.Reader
	timeout        time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewTimeoutReader(reader io.Reader, initialTimeout bool, timeout time.Duration) *TimeoutReader {
	timeoutReader := &TimeoutReader{
		initialTimeout: initialTimeout,
		canSet:         CanSetReadTimeout(reader),
		reader:         reader,
		timeout:        timeout,
	}

	if timeoutReader.timeout == 0 {
		return timeoutReader
	}

	if !timeoutReader.canSet && initialTimeout {
		timeoutReader.ctx, timeoutReader.cancel = context.WithTimeout(context.Background(), timeout)
	}

	return timeoutReader
}

func (timeoutReader *TimeoutReader) Read(p []byte) (int, error) {
	if timeoutReader.timeout == 0 {
		return timeoutReader.reader.Read(p)
	}

	if !timeoutReader.initialTimeout && timeoutReader.FirstRead.IsZero() {
		n, err := timeoutReader.reader.Read(p[0:1])

		timeoutReader.FirstRead = time.Now()

		return n, err
	}

	if timeoutReader.canSet {
		n, err := ReadWithTimeout(timeoutReader.reader, timeoutReader.timeout, p)

		if timeoutReader.FirstRead.IsZero() {
			timeoutReader.FirstRead = time.Now()
		}

		return n, err
	}

	if !timeoutReader.initialTimeout {
		timeoutReader.ctx, timeoutReader.cancel = context.WithTimeout(context.Background(), timeoutReader.timeout)
	}

	r := ctxio.NewReader(timeoutReader.ctx, timeoutReader.reader)

	n, err := r.Read(p)

	if timeoutReader.FirstRead.IsZero() {
		timeoutReader.FirstRead = time.Now()
	}

	if !timeoutReader.initialTimeout || IsErrTimeout(err) {
		timeoutReader.cancel()
	}

	return n, err
}

type TimeoutWriter struct {
	FirstWrite     time.Time
	initialTimeout bool
	canSet         bool
	writer         io.Writer
	timeout        time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewTimeoutWriter(writer io.Writer, initialTimeout bool, timeout time.Duration) *TimeoutWriter {
	timeoutWriter := &TimeoutWriter{
		initialTimeout: initialTimeout,
		canSet:         CanSetWriteTimeout(writer),
		writer:         writer,
		timeout:        timeout,
	}

	if timeoutWriter.timeout == 0 {
		return timeoutWriter
	}

	if !timeoutWriter.canSet && initialTimeout {
		timeoutWriter.ctx, timeoutWriter.cancel = context.WithTimeout(context.Background(), timeout)
	}

	return timeoutWriter
}

func (timeoutWriter *TimeoutWriter) Write(p []byte) (int, error) {
	if timeoutWriter.timeout == 0 {
		return timeoutWriter.writer.Write(p)
	}

	if !timeoutWriter.initialTimeout && timeoutWriter.FirstWrite.IsZero() {
		n, err := timeoutWriter.writer.Write(p[0:1])

		timeoutWriter.FirstWrite = time.Now()

		return n, err
	}

	if timeoutWriter.canSet {
		n, err := WriteWithTimeout(timeoutWriter.writer, timeoutWriter.timeout, p)

		if timeoutWriter.FirstWrite.IsZero() {
			timeoutWriter.FirstWrite = time.Now()
		}

		return n, err
	}

	if !timeoutWriter.initialTimeout {
		timeoutWriter.ctx, timeoutWriter.cancel = context.WithTimeout(context.Background(), timeoutWriter.timeout)
	}

	w := ctxio.NewWriter(timeoutWriter.ctx, timeoutWriter.writer)

	n, err := w.Write(p)

	if timeoutWriter.FirstWrite.IsZero() {
		timeoutWriter.FirstWrite = time.Now()
	}

	if !timeoutWriter.initialTimeout || IsErrTimeout(err) {
		timeoutWriter.cancel()
	}

	return n, err
}

type hasReadDeadline interface {
	SetReadDeadline(t time.Time) error
}

type hasReadTimeout interface {
	SetReadTimeout(t time.Duration) error
}

type hasWriteDeadline interface {
	SetWriteDeadline(t time.Time) error
}

func CanSetReadTimeout(reader io.Reader) bool {
	_, ok := reader.(hasReadDeadline)
	if ok {
		return true
	}

	_, ok = reader.(hasReadTimeout)
	if ok {
		return true
	}

	return false
}

func CanSetWriteTimeout(writer io.Writer) bool {
	_, ok := writer.(hasWriteDeadline)
	if ok {
		return true
	}

	return false
}

func SetReadTimeout(reader io.Reader, timeout time.Duration) error {
	deadliner, ok := reader.(hasReadDeadline)
	if ok {
		err := deadliner.SetReadDeadline(time.Now().Add(timeout))
		if Error(err) {
			return err
		}

		return nil
	}

	timeouter, ok := reader.(hasReadTimeout)
	if ok {
		err := timeouter.SetReadTimeout(timeout)
		if Error(err) {
			return err
		}

		return nil
	}

	return TraceError(&ErrSetTimeout{
		Mode: "READ",
	})
}

func SetWriteTimeout(writer io.Writer, timeout time.Duration) error {
	deadliner, ok := writer.(hasWriteDeadline)
	if ok {
		err := deadliner.SetWriteDeadline(time.Now().Add(timeout))
		if Error(err) {
			return err
		}

		return nil
	}

	return TraceError(&ErrSetTimeout{
		Mode: "WRITE",
	})
}

func ReadWithTimeout(reader io.Reader, timeout time.Duration, ba []byte) (int, error) {
	isDeadlineSet := false

	if timeout > 0 {
		err := SetReadTimeout(reader, timeout)
		if Error(err) {
			return 0, err
		}

		isDeadlineSet = true
	}

	start := time.Now()

	n, err := reader.Read(ba)

	if err == nil && isDeadlineSet && time.Since(start) >= timeout && n == 0 {
		return n, &ErrTimeout{
			Duration: timeout,
		}
	}

	return n, err
}

func WriteWithTimeout(writer io.Writer, timeout time.Duration, ba []byte) (int, error) {
	isDeadlineSet := false

	if timeout > 0 {
		err := SetWriteTimeout(writer, timeout)
		if Error(err) {
			return 0, err
		}

		isDeadlineSet = true
	}

	start := time.Now()

	n, err := writer.Write(ba)

	if err == nil && isDeadlineSet && time.Since(start) >= timeout && n == 0 {
		return n, &ErrTimeout{
			Duration: timeout,
		}
	}

	return n, err
}

func WriteFully(w io.Writer, p []byte) (int, error) {
	sum := 0

	for {
		n, err := w.Write(p)
		sum += n

		if Error(err) {
			return sum, err
		}

		if n == len(p) {
			return sum, nil
		}

		p = p[n:]
	}
}

func ReadFully(r io.Reader, p []byte) (int, error) {
	sum := 0

	for {
		n, err := r.Read(p)
		sum += n

		if Error(err) {
			return sum, err
		}

		if n == len(p) {
			return sum, nil
		}

		p = p[n:]
	}
}
