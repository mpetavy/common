package common

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gookit/color"
	"github.com/kardianos/service"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	LEVEL_PROLOG = iota
	LEVEL_DEBUG
	LEVEL_INFO
	LEVEL_WARN
	LEVEL_ERROR
	LEVEL_PANIC
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

type logEntry struct {
	levelInt int    `json:"levelInt"`
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
	WriteString(int, string)
	Logs(io.Writer) error
	Close()
}

type memoryWriter struct {
	lines []string
}

func (this *memoryWriter) WriteString(level int, txt string) {
	if level == LEVEL_PROLOG {
		return
	}

	if len(this.lines) == 1000 {
		this.lines = this.lines[1:]
	}

	this.lines = append(this.lines, txt)
}

func (this *memoryWriter) Logs(w io.Writer) error {
	for _, l := range this.lines {
		_, err := w.Write([]byte(l))

		if Error(err) {
			return err
		}
	}

	return nil
}

func (this *memoryWriter) Close() {
}

func newMemoryWriter() *memoryWriter {
	writer := memoryWriter{
		lines: make([]string, 0),
	}

	return &writer
}

type fileWriter struct {
	filesize int64
	file     *os.File
}

func (this *fileWriter) WriteString(level int, txt string) {
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

	this.filesize += int64(len(ba))
}

func (this *fileWriter) Logs(w io.Writer) error {
	for i := *FlagIoFileBackups; i >= 0; i-- {
		var src string

		if *FlagIoFileBackups == 1 {
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
		if Error(err) {
			return err
		}
	}

	return nil
}

func (this *fileWriter) Close() {
	if this.file != nil {
		Ignore(this.file.Close())
		this.file = nil
	}
}

func newFileWriter() (*fileWriter, error) {
	filesize := int64(0)

	if FileExists(realLogFilename()) {
		var err error

		filesize, err = FileSize(realLogFilename())
		if Error(err) {
			return nil, err
		}
	}

	logFile, err := os.OpenFile(realLogFilename(), os.O_RDWR|os.O_CREATE|os.O_APPEND, DefaultFileMode)
	if Error(err) {
		return nil, err
	}

	writer := fileWriter{
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
	case LEVEL_PANIC:
		return "PANIC"
	default:
		return "INFO"
	}
}

type redirectGoLogger struct {
	io.Writer
}

func (r *redirectGoLogger) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))

	c := 0
	for len(msg) > 0 {
		p := strings.Index(msg, " ")
		if p == -1 {
			return 0, nil
		}

		msg = msg[p+1:]
		c++

		if c == 2 {
			break
		}

	}

	err := fmt.Errorf(msg)

	if IsSuppressedError(err) {
		if *FlagLogVerbose {
			DebugError(err)
		}
	} else {
		Error(err)
	}

	return len(p), nil
}

func initLog() {
	DebugFunc()

	if realLogFilename() != "" {
		var err error

		logger, err = newFileWriter()
		Error(err)
	}

	if logger == nil {
		logger = newMemoryWriter()
	}

	log.SetOutput(&redirectGoLogger{})

	if *FlagLogSys && !IsRunningInteractive() {
		systemLoggerCh = make(chan error, 5)

		var err error

		systemLogger, err = app.Service.Logger(systemLoggerCh)
		Error(err)
	}

	if app != nil {
		prolog(fmt.Sprintf(">>> Start - %s %s %s", strings.ToUpper(app.Name), app.Version, strings.Repeat("-", 98)))
		prolog(fmt.Sprintf(">>> Cmdline : %s", strings.Join(SurroundWith(os.Args, "\""), " ")))
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

	appendLog(LEVEL_PROLOG, GetRuntimeInfo(1), t, nil)
}

// Debug prints out the information
func Debug(t string, arg ...interface{}) {
	if FlagLogVerbose == nil || !*FlagLogVerbose {
		return
	}

	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	appendLog(LEVEL_DEBUG, GetRuntimeInfo(1), t, nil)
}

// DebugError prints out the error
func DebugError(err error) bool {
	if FlagLogVerbose == nil || !*FlagLogVerbose {
		return err != nil
	}

	if err != nil && !IsErrExit(err) {
		ri := GetRuntimeInfo(1)

		appendLog(LEVEL_DEBUG, ri, fmt.Sprintf("Error: %s", errorString(ri, err)), nil)
	}

	return err != nil
}

// Info prints out the information
func Info(t string, arg ...interface{}) {
	if FlagLogVerbose == nil {
		return
	}

	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	appendLog(LEVEL_INFO, GetRuntimeInfo(1), t, nil)
}

// Warn prints out the information
func Warn(t string, arg ...interface{}) {
	if FlagLogVerbose == nil {
		return
	}

	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	appendLog(LEVEL_WARN, GetRuntimeInfo(1), t, nil)
}

func WarnError(err error) bool {
	if FlagLogVerbose == nil {
		return err != nil
	}

	if err != nil && !IsErrExit(err) {
		ri := GetRuntimeInfo(1)

		appendLog(LEVEL_WARN, ri, fmt.Sprintf("Error: %s", errorString(ri, err)), nil)
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
	if FlagLogVerbose == nil || !*FlagLogVerbose {
		return
	}

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

	appendLog(LEVEL_DEBUG, ri, t, nil)
}

// Ignore just ignores the error
func Ignore(arg ...interface{}) bool {
	b := false

	if len(arg) > 0 {
		_, b = arg[len(arg)-1].(error)
	}

	return b
}

func Panic(err error) {
	if err == nil {
		return
	}

	ri := GetRuntimeInfo(1)

	appendLog(LEVEL_PANIC, ri, errorString(ri, err), err)

	Exit(1)
}

func Error(err error) bool {
	if FlagLogVerbose == nil {
		return err != nil
	}

	if err != nil && !IsErrExit(err) {
		ri := GetRuntimeInfo(1)

		appendLog(LEVEL_ERROR, ri, errorString(ri, err), err)
	}

	return err != nil
}

func ErrorReturn(err error) error {
	if FlagLogVerbose == nil {
		return err
	}

	if err != nil && !IsErrExit(err) {
		ri := GetRuntimeInfo(1)

		appendLog(LEVEL_ERROR, ri, errorString(ri, err), err)
	}

	return err
}

func appendLog(level int, ri RuntimeInfo, msg string, err error) {
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

	entry := logEntry{
		levelInt: level,
		Level:    levelToString(level),
		Clock:    time.Now().Format(DateTimeMilliMask),
		Runtime:  ri.String(),
		Msg:      Capitalize(strings.TrimRight(strings.TrimSpace(msg), "\r\n")),
	}

	s := entry.String(*FlagLogJson, *FlagLogVerbose)

	if logger != nil {
		logger.WriteString(entry.levelInt, fmt.Sprintf("%s\n", s))
	}

	if level != LEVEL_PROLOG {
		if *FlagLogVerbose {
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
			case LEVEL_PANIC:
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
		} else {
			fmt.Printf("%s\n", s)
		}

		if *FlagLogSys && systemLogger != nil {
			switch entry.levelInt {
			case LEVEL_WARN:
				Error(systemLogger.Warning(entry.String(false, false)))
			case LEVEL_ERROR:
				Error(systemLogger.Error(entry.String(false, false)))
			case LEVEL_PANIC:
				Error(systemLogger.Error(fmt.Sprintf("PANIC: %s", entry.String(false, false))))
			case LEVEL_DEBUG:
				fallthrough
			case LEVEL_INFO:
				Error(systemLogger.Info(entry.String(false, false)))
			}
		}
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
