package common

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gookit/color"
	"github.com/kardianos/service"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DB level
	LEVEL_FILE = iota
	// LEVEL_DEBUG level
	LEVEL_DEBUG
	// LEVEL_INFO level
	LEVEL_INFO
	// LEVEL_ERROR level
	LEVEL_WARN
	// LEVEL_ERROR level
	LEVEL_ERROR
)

var (
	FlagLogVerbose     *bool
	FlagLogIO          *bool
	FlagLogFileName    *string
	FlagLogFileSize    *int64
	FlagLogJson        *bool
	FlagLogSys         *bool
	logger             logWriter
	defaultLogFilename string
	mu                 sync.Mutex
	lastErr            string
	lastErrTime        time.Time
	systemLoggerCh     chan<- error
	systemLogger       service.Logger
	gotest             goTesting
)

const (
	FlagNameLogFileName = "log.file"
	FlagNameLogFileSize = "log.filesize"
	FlagNameLogVerbose  = "log.verbose"
	FlagNameLogIO       = "log.io"
	FlagNameLogJson     = "log.json"
	FlagNameLogSys      = "log.sys"
)

type ErrExit struct {
}

func init() {
	defaultLogFilename = CleanPath(AppFilename(".log"))

	FlagLogFileName = flag.String(FlagNameLogFileName, "", fmt.Sprintf("filename to log logFile (use \".\" for %s)", defaultLogFilename))
	FlagLogFileSize = flag.Int64(FlagNameLogFileSize, 5*1024*1024, "max log file size")
	FlagLogVerbose = flag.Bool(FlagNameLogVerbose, false, "verbose logging")
	FlagLogIO = flag.Bool(FlagNameLogIO, false, "trace logging")
	FlagLogJson = flag.Bool(FlagNameLogJson, false, "JSON output")
	FlagLogSys = flag.Bool(FlagNameLogSys, false, "Use OS system logger")
}

func InitTesting(v goTesting) {
	gotest = v
}

type goTesting interface {
	Logf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

func (e *ErrExit) Error() string { return "" }

type logEntry struct {
	levelInt int
	Clock    string `json:"clock"`
	Level    string `json:"level"`
	Runtime  string `json:"runtime"`
	Msg      string `json:"msg"`
}

func (l *logEntry) String(jsn bool, verbose bool) string {
	if jsn {
		ba, _ := json.Marshal(l)

		return string(ba)
	} else {
		if verbose {
			return fmt.Sprintf("%s %s %-40.40s %s", l.Clock, FillString(l.Level, 5, false, " "), l.Runtime, l.Msg)
		} else {
			return l.Msg
		}
	}
}

type logWriter interface {
	WriteString(txt string)
	Logs(io.Writer) error
	Close()
}

type logMemoryWriter struct {
	lines []string
}

func (this *logMemoryWriter) WriteString(txt string) {
	if len(this.lines) == 1000 {
		this.lines = this.lines[1:]
	}

	this.lines = append(this.lines, txt)
}

func (this *logMemoryWriter) Logs(w io.Writer) error {
	for _, l := range this.lines {
		_, err := w.Write([]byte(l))

		if err != nil {
			return err
		}
	}

	return nil
}

func (this *logMemoryWriter) Close() {
}

func newLogMemoryWriter() *logMemoryWriter {
	writer := logMemoryWriter{
		lines: make([]string, 0),
	}

	return &writer
}

type logFileWriter struct {
	filesize int64
	file     *os.File
}

func (this *logFileWriter) WriteString(txt string) {
	if this.file == nil {
		return
	}

	if this.filesize >= *FlagLogFileSize {
		this.filesize = 0

		if this.file != nil {
			Ignore(this.file.Close())
			this.file = nil
		}

		Ignore(FileBackup(realLogFilename()))

		this.file, _ = os.OpenFile(realLogFilename(), os.O_RDWR|os.O_CREATE|os.O_APPEND, DefaultFileMode)
	}

	if this.file == nil {
		return
	}

	ba := []byte(txt)

	Ignore(this.file.Write(ba))
	Ignore(this.file.Sync())

	this.filesize += int64(len(ba))
}

func (this *logFileWriter) Logs(w io.Writer) error {
	for i := *FlagCountBackups; i >= 0; i-- {
		var src string

		if *FlagCountBackups == 1 {
			src = realLogFilename() + ".bak"

			if !fileExists(src) {
				src = ""
			}
		}

		if src == "" {
			if i > 0 {
				src = realLogFilename() + "." + strconv.Itoa(i)
			} else {
				src = realLogFilename()
			}
		}

		file, err := os.Open(src)
		if err != nil {
			continue
		}

		_, err = io.Copy(w, file)
		_ = file.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *logFileWriter) Close() {
	if this.file != nil {
		Ignore(this.file.Close())
		this.file = nil
	}
}

func newLogFileWriter() (*logFileWriter, error) {
	filesize := int64(0)

	if FileExists(realLogFilename()) {
		var err error

		filesize, err = FileSize(realLogFilename())
		if err != nil {
			return nil, err
		}
	}

	logFile, err := os.OpenFile(realLogFilename(), os.O_RDWR|os.O_CREATE|os.O_APPEND, DefaultFileMode)
	if err != nil {
		return nil, err
	}

	writer := logFileWriter{
		file:     logFile,
		filesize: filesize,
	}

	return &writer, nil
}

func levelToString(level int) string {
	switch level {
	case LEVEL_DEBUG:
		return "DEBUG"
	case LEVEL_INFO:
		return "INFO"
	case LEVEL_WARN:
		return "WARN"
	case LEVEL_ERROR:
		return "ERROR"
	default:
		return "INFO"
	}
}

func initLog() {
	DebugFunc()

	if realLogFilename() != "" {
		var err error

		logger, err = newLogFileWriter()
		if err != nil {
			Error(err)
		}
	}

	if logger == nil {
		logger = newLogMemoryWriter()
	}

	if *FlagLogSys && !IsRunningInteractive() {
		systemLoggerCh = make(chan error, 5)

		var err error

		systemLogger, err = app.Service.Logger(systemLoggerCh)
		if err != nil {
			Error(err)
		}
	}

	if app != nil {
		prolog(fmt.Sprintf(">>> Start - %s %s %s", strings.ToUpper(app.Name), app.Version, strings.Repeat("-", 98)))
		prolog(fmt.Sprintf(">>> Cmdline : %s", strings.Join(SurroundWith(os.Args, "\""), " ")))
	}
}

func writeEntry(entry logEntry) {
	s := entry.String(*FlagLogJson, *FlagLogVerbose)

	if entry.levelInt != LEVEL_FILE {
		if !*FlagLogVerbose {
			if gotest != nil {
				gotest.Logf(s)
			} else {
				fmt.Println(s)
			}

			return
		}

		switch entry.levelInt {
		case LEVEL_WARN:
			if gotest != nil {
				gotest.Logf(s)
			} else {
				color.Warn.Println(s)
			}
		case LEVEL_ERROR:
			if gotest != nil {
				gotest.Fatalf(s)
			} else {
				color.Error.Println(s)
			}
		default:
			if gotest != nil {
				gotest.Logf(s)
			} else {
				fmt.Printf("%s\n", s)
			}
		}
	}

	if logger != nil {
		logger.WriteString(fmt.Sprintf("%s\n", s))
	}

	if *FlagLogSys && systemLogger != nil {
		switch entry.levelInt {
		case LEVEL_WARN:
			Error(systemLogger.Warning(entry.String(false, false)))
		case LEVEL_ERROR:
			Error(systemLogger.Error(entry.String(false, false)))
		case LEVEL_DEBUG:
			fallthrough
		case LEVEL_INFO:
			Error(systemLogger.Info(entry.String(false, false)))
		}
	}
}

func closeLogFile() {
	prolog(fmt.Sprintf("<<< End - %s %s %s", strings.ToUpper(app.Name), app.Version, strings.Repeat("-", 100)))

	if logger != nil {
		logger.Close()
	}
}

// logFile prints out the information
func prolog(t string, arg ...interface{}) {
	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_FILE, GetRuntimeInfo(1), t, nil)
}

// Debug prints out the information
func Debug(t string, arg ...interface{}) {
	if *FlagLogVerbose {
		if len(arg) > 0 {
			t = fmt.Sprintf(t, arg...)
		}

		log(LEVEL_DEBUG, GetRuntimeInfo(1), t, nil)
	}
}

// DebugError prints out the error
func DebugError(err error) bool {
	if *FlagLogVerbose {
		if err != nil && !IsErrExit(err) {
			ri := GetRuntimeInfo(1)

			log(LEVEL_DEBUG, ri, fmt.Sprintf("Error: %s", errorString(ri, err)), nil)
		}
	}

	return err != nil
}

// Info prints out the information
func Info(t string, arg ...interface{}) {
	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_INFO, GetRuntimeInfo(1), t, nil)
}

// Warn prints out the information
func Warn(t string, arg ...interface{}) {
	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_WARN, GetRuntimeInfo(1), t, nil)
}

func WarnError(err error) bool {
	if err != nil && !IsErrExit(err) {
		ri := GetRuntimeInfo(1)

		log(LEVEL_WARN, ri, fmt.Sprintf("Error: %s", errorString(ri, err)), nil)
	}

	return err != nil
}

func errorString(ri RuntimeInfo, err error) string {
	if *FlagLogVerbose {
		return fmt.Sprintf("%s [%T]\n%s", err.Error(), err, ri.Stack)
	}

	return fmt.Sprintf("Error: %s", err.Error())
}

// DebugFunc prints out the current executon func
func DebugFunc(arg ...interface{}) {
	ri := GetRuntimeInfo(1)

	t := ri.Fn + "()"

	if len(arg) == 1 {
		t = fmt.Sprintf("%s: %v", t, arg[0])
	}
	if len(arg) > 1 {
		s, ok := arg[0].(string)

		if ok {
			t = fmt.Sprintf("%s: %s", t, fmt.Sprintf(s, arg[1:]...))
		} else {
			t = fmt.Sprintf("%s: %s", t, fmt.Sprintf("%v", arg[1:]...))
		}
	}

	log(LEVEL_DEBUG, ri, t, nil)
}

// Ignore just ignores the error
func Ignore(arg ...interface{}) bool {
	b := false

	if len(arg) > 0 {
		_, b = arg[len(arg)-1].(error)
	}

	return b
}

// Error prints out the error
func Error(err error) bool {
	if err != nil && !IsErrExit(err) {
		ri := GetRuntimeInfo(1)

		log(LEVEL_ERROR, ri, errorString(ri, err), err)
	}

	return err != nil
}

func ErrorReturn(err error) error {
	if err != nil && !IsErrExit(err) {
		ri := GetRuntimeInfo(1)

		log(LEVEL_ERROR, ri, errorString(ri, err), err)
	}

	return err
}

func log(level int, ri RuntimeInfo, msg string, err error) {
	mu.Lock()
	defer mu.Unlock()

	if level == LEVEL_ERROR {
		defer func() {
			lastErr = err.Error()
			lastErrTime = time.Now()
		}()

		if err.Error() == lastErr && time.Since(lastErrTime) < time.Millisecond*100 {
			return
		}
	}

	if level == LEVEL_FILE || (FlagLogVerbose != nil && *FlagLogVerbose) || level > LEVEL_DEBUG {
		le := logEntry{
			levelInt: level,
			Level:    levelToString(level),
			Clock:    time.Now().Format(DateTimeMilliMask),
			Runtime:  ri.String(),
			Msg:      Capitalize(strings.TrimRight(strings.TrimSpace(msg), "\r\n")),
		}

		writeEntry(le)
	}
}

func CmdToString(cmd *exec.Cmd) string {
	s := SurroundWith(cmd.Args, "\"")

	return strings.Join(s, " ")
}

func Logs(w io.Writer) error {
	if logger == nil {
		return fmt.Errorf("no logger")
	}

	return logger.Logs(w)
}

func LogsAvailable() bool {
	return logger != nil
}

func realLogFilename() string {
	if *FlagLogFileName == "." {
		return defaultLogFilename
	} else {
		return *FlagLogFileName
	}
}
