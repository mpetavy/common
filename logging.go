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
	FlagLogFileSize    *int
	FlagLogJson        *bool
	FlagLogSys         *bool
	FlagLogCount       *int
	logger             logWriter
	defaultLogFilename string
	mu                 sync.Mutex
	lastErr            string
	syslogLoggerCh     chan<- error
	syslogLogger       service.Logger
	gotest             goTesting

	ColorDefault color.Color = 0
	ColorDebug   color.Color = ColorDefault
	ColorInfo    color.Color = ColorDefault
	ColorWarn    color.Color = color.Yellow
	ColorError   color.Color = color.Red
	ColorPanic   color.Color = color.Red
)

const (
	FlagNameLogFileName = "log.file"
	FlagNameLogFileSize = "log.filesize"
	FlagNameLogVerbose  = "log.verbose"
	FlagNameLogIO       = "log.io"
	FlagNameLogJson     = "log.json"
	FlagNameLogSys      = "log.sys"
	FlagNameLogCount    = "log.count"
)

func init() {
	defaultLogFilename = CleanPath(AppFilename(".log"))

	FlagLogFileName = flag.String(FlagNameLogFileName, "", fmt.Sprintf("filename to log logFile (use \".\" for %s)", defaultLogFilename))
	FlagLogFileSize = flag.Int(FlagNameLogFileSize, 5*1024*1024, "max log file size")
	FlagLogVerbose = flag.Bool(FlagNameLogVerbose, false, "verbose logging")
	FlagLogIO = flag.Bool(FlagNameLogIO, false, "trace logging")
	FlagLogJson = flag.Bool(FlagNameLogJson, false, "JSON output")
	FlagLogSys = flag.Bool(FlagNameLogSys, false, "Use OS system logger")
	FlagLogCount = flag.Int(FlagNameLogCount, 1000, "log count")
}

func InitTesting(v goTesting) {
	gotest = v
}

type goTesting interface {
	Logf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

type logEntry struct {
	levelInt int
	color    color.Color
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
	GetLogs(io.Writer) error
	ClearLogs() error
	Close()
}

type memoryWriter struct {
	lines []string
}

func (this *memoryWriter) WriteString(level int, txt string) {
	if level == LEVEL_PROLOG {
		return
	}

	if len(this.lines) >= *FlagLogCount {
		start := len(this.lines) - *FlagLogCount + 1

		this.lines = this.lines[start:]
	}

	this.lines = append(this.lines, txt)
}

func (this *memoryWriter) ClearLogs() error {
	mu.Lock()
	defer mu.Unlock()

	this.lines = this.lines[:0]

	return nil
}

func (this *memoryWriter) GetLogs(w io.Writer) error {
	mu.Lock()
	defer mu.Unlock()

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

	if this.filesize >= int64(*FlagLogFileSize) {
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

func (this fileWriter) ClearLogs() error {
	mu.Lock()
	defer mu.Unlock()

	mu.Lock()
	defer mu.Unlock()

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

		if FileExists(src) {
			err := FileDelete(src)
			if Error(err) {
				return err
			}
		}
	}

	return nil
}

func (this *fileWriter) GetLogs(w io.Writer) error {
	mu.Lock()
	defer mu.Unlock()

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
		return "debug"
	case LEVEL_INFO:
		return "info"
	case LEVEL_WARN:
		return "warn"
	case LEVEL_ERROR:
		return "error"
	case LEVEL_PANIC:
		return "panic"
	default:
		return "info"
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

	Error(fmt.Errorf(msg))

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
		syslogLoggerCh = make(chan error, 5)

		var err error

		syslogLogger, err = app.Service.Logger(syslogLoggerCh)
		Error(err)
	}

	if app != nil {
		prolog(fmt.Sprintf(">>> Start - %s %s %s", strings.ToUpper(app.Name), app.Version, strings.Repeat("-", 98)))
		prolog(fmt.Sprintf(">>> Cmdline : %s", strings.Join(SurroundWith(os.Args, "\""), " ")))
	}
}

func closeLog() {
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

	appendLog(LEVEL_PROLOG, ColorDefault, GetRuntimeInfo(1), t, nil)
}

// Debug prints out the information
func Debug(t string, arg ...interface{}) {
	if FlagLogVerbose == nil || !*FlagLogVerbose {
		return
	}

	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	appendLog(LEVEL_DEBUG, ColorDebug, GetRuntimeInfo(1), t, nil)
}

// DebugError prints out the error
func DebugError(err error) bool {
	if FlagLogVerbose == nil || !*FlagLogVerbose {
		return err != nil
	}

	if err != nil && !IsErrExit(err) && !IsSuppressedError(err) {
		ri := GetRuntimeInfo(1)

		appendLog(LEVEL_DEBUG, ColorDebug, ri, errorString(LEVEL_ERROR, ri, err), nil)
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

	appendLog(LEVEL_INFO, ColorInfo, GetRuntimeInfo(1), t, nil)
}

// Warn prints out the information
func Warn(t string, arg ...interface{}) {
	if FlagLogVerbose == nil {
		return
	}

	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	appendLog(LEVEL_WARN, ColorWarn, GetRuntimeInfo(1), t, nil)
}

func WarnError(err error) bool {
	if FlagLogVerbose == nil {
		return err != nil
	}

	if err != nil && !IsErrExit(err) && !IsSuppressedError(err) {
		ri := GetRuntimeInfo(1)

		appendLog(LEVEL_WARN, ColorWarn, ri, errorString(LEVEL_ERROR, ri, err), nil)
	}

	return err != nil
}

func errorString(level int, ri RuntimeInfo, err error) string {
	if *FlagLogVerbose {
		return fmt.Sprintf("%s [%T]\n%s", err.Error(), err, ri.Stack)
	}

	return fmt.Sprintf("%s: %s", Capitalize(levelToString(level)), err.Error())
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

	appendLog(LEVEL_DEBUG, ColorDebug, ri, t, nil)
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

	appendLog(LEVEL_PANIC, ColorPanic, ri, errorString(LEVEL_PANIC, ri, err), err)

	Exit(1)
}

func Error(err error) bool {
	if FlagLogVerbose == nil {
		return err != nil
	}

	if err != nil && !IsErrExit(err) && !IsSuppressedError(err) {
		ri := GetRuntimeInfo(1)

		appendLog(LEVEL_ERROR, ColorError, ri, errorString(LEVEL_ERROR, ri, err), err)
	}

	return err != nil
}

func appendLog(level int, color color.Color, ri RuntimeInfo, msg string, err error) {
	mu.Lock()
	defer mu.Unlock()

	if level >= LEVEL_ERROR {
		if err.Error() == lastErr {
			return
		}

		lastErr = err.Error()
	} else {
		lastErr = ""
	}

	entry := logEntry{
		levelInt: level,
		color:    color,
		Level:    strings.ToUpper(levelToString(level)),
		Clock:    time.Now().Format(DateTimeMilliMask),
		Runtime:  ri.String(),
		Msg:      Capitalize(strings.TrimRight(strings.TrimSpace(msg), "\r\n")),
	}

	s := entry.String(*FlagLogJson, *FlagLogVerbose)

	// fileLogger or memoryLogger
	if logger != nil {
		logger.WriteString(entry.levelInt, fmt.Sprintf("%s\n", entry.String(*FlagLogJson, true)))
	}

	if level != LEVEL_PROLOG {
		if syslogLogger != nil {
			switch entry.levelInt {
			case LEVEL_WARN:
				Error(syslogLogger.Warning(entry.Msg))
			case LEVEL_ERROR:
				Error(syslogLogger.Error(entry.Msg))
			case LEVEL_PANIC:
				Error(syslogLogger.Error(entry.Msg))
			case LEVEL_DEBUG:
				fallthrough
			case LEVEL_INFO:
				Error(syslogLogger.Info(entry.Msg))
			}
		}

		if gotest != nil {
			gotest.Logf(s)
		} else {
			if entry.color != ColorDefault {
				entry.color.Println(s)
			} else {
				fmt.Println(s)
			}
		}
	}
}

func CmdToString(cmd *exec.Cmd) string {
	s := SurroundWith(cmd.Args, "\"")

	return strings.Join(s, " ")
}

func GetLogs(w io.Writer) error {
	if logger == nil {
		return fmt.Errorf("no logger")
	}

	return logger.GetLogs(w)
}

func ClearLogs() error {
	if logger == nil {
		return fmt.Errorf("no logger")
	}

	return logger.ClearLogs()
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
