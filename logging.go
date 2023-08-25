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
	"reflect"
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
	FlagLogFileName = flag.String(FlagNameLogFileName, "", "filename to log file")
	FlagLogFileSize = flag.Int(FlagNameLogFileSize, 5*1024*1024, "max log file size")
	FlagLogVerbose  = flag.Bool(FlagNameLogVerbose, false, "verbose logging")
	FlagLogIO       = flag.Bool(FlagNameLogIO, false, "trace logging")
	FlagLogJson     = flag.Bool(FlagNameLogJson, false, "JSON output")
	FlagLogSys      = flag.Bool(FlagNameLogSys, false, "Use OS system logger")
	FlagLogCount    = flag.Int(FlagNameLogCount, 1000, "log count")
	FlagLogBreak    = flag.Bool(FlagNameLogBreak, false, "break on error")

	logger         logWriter
	mu             ReentrantMutex
	lastErr        string
	syslogLoggerCh chan<- error
	syslogLogger   service.Logger
	gotest         goTesting
	logCh          = make(chan logEntry)
	wgLogCh        sync.WaitGroup
	isLogClosed    bool

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
	FlagNameLogBreak    = "log.break"
)

func init() {
	Events.AddListener(EventShutdown{}, func(event Event) {
		Error(closeLog())
	})

	Events.AddListener(EventFlagsParsed{}, func(event Event) {
		if *FlagLogSys && IsLinux() && !IsRunningInteractive() {
			// with SYSTEMD everything which is printed to console is automatically printed to journalctl

			*FlagLogSys = false
		}
	})

	wgLogCh.Add(1)

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine(1))
		defer wgLogCh.Done()

		for {
			entry, ok := <-logCh
			if !ok {
				return
			}

			logOutput(entry)
		}
	}()
}

func InitTesting(t goTesting) {
	gotest = t
}

type goTesting interface {
	Logf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type logEntry struct {
	levelInt int
	color    color.Color
	Clock    string `json:"clock"`
	Level    string `json:"level"`
	Runtime  string `json:"runtime"`
	Msg      string `json:"msg"`
}

func (l *logEntry) toString(asJson bool, verbose bool) string {
	if asJson {
		ba, err := json.Marshal(l)
		loggingError(err)

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
	writeString(int, string)
	getLogs(io.Writer) error
	clearLogs() error
	close()
}

type memoryWriter struct {
	lines []string
}

func (this *memoryWriter) writeString(level int, txt string) {
	if level == LEVEL_PROLOG {
		return
	}

	if len(this.lines) >= *FlagLogCount {
		start := len(this.lines) - *FlagLogCount + 1

		this.lines = this.lines[start:]
	}

	this.lines = append(this.lines, txt)
}

func (this *memoryWriter) clearLogs() error {
	mu.Lock()
	defer mu.Unlock()

	this.lines = this.lines[:0]

	return nil
}

func (this *memoryWriter) getLogs(w io.Writer) error {
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

func (this *memoryWriter) close() {
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

func (this *fileWriter) writeString(level int, txt string) {
	if this.file == nil {
		return
	}

	if this.filesize >= int64(*FlagLogFileSize) {
		this.filesize = 0

		if this.file != nil {
			loggingError(this.file.Close())
			this.file = nil
		}

		loggingError(FileBackup(*FlagLogFileName))

		var err error

		this.file, err = os.OpenFile(*FlagLogFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, DefaultFileMode)

		loggingError(err)
	}

	ba := []byte(txt)

	n, err := this.file.Write(ba)
	loggingError(err)

	this.filesize += int64(n)
}

func (this fileWriter) clearLogs() error {
	mu.Lock()
	defer mu.Unlock()

	for i := *FlagIoFileBackups; i >= 0; i-- {
		var src string

		if *FlagIoFileBackups == 1 {
			src = *FlagLogFileName + ".bak"

			if !FileExists_(src) {
				src = ""
			}
		}

		if src == "" {
			if i > 0 {
				src = *FlagLogFileName + "." + strconv.Itoa(i)
			} else {
				src = *FlagLogFileName
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

func (this *fileWriter) getLogs(w io.Writer) error {
	mu.Lock()
	defer mu.Unlock()

	for i := *FlagIoFileBackups; i >= 0; i-- {
		var src string

		if *FlagIoFileBackups == 1 {
			src = *FlagLogFileName + ".bak"

			if !FileExists_(src) {
				src = ""
			}
		}

		if src == "" {
			if i > 0 {
				src = *FlagLogFileName + "." + strconv.Itoa(i)
			} else {
				src = *FlagLogFileName
			}
		}

		if !FileExists_(src) {
			continue
		}

		file, err := os.Open(src)
		if loggingError(err) {
			continue
		}

		_, err = io.Copy(w, file)
		if loggingError(err) {
			continue
		}
		err = file.Close()
		if loggingError(err) {
			continue
		}
	}

	return nil
}

func (this *fileWriter) close() {
	mu.Lock()
	defer mu.Unlock()

	if this.file != nil {
		loggingError(this.file.Close())
		this.file = nil
	}
}

func newFileWriter() (*fileWriter, error) {
	filesize := int64(0)

	if FileExists(*FlagLogFileName) {
		var err error

		filesize, err = FileSize(*FlagLogFileName)
		if Error(err) {
			return nil, err
		}
	}

	logFile, err := os.OpenFile(*FlagLogFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, DefaultFileMode)
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

func initLog() error {
	DebugFunc()

	if *FlagLogFileName != "" {
		var err error

		logger, err = newFileWriter()
		Error(err)
	}

	if logger == nil {
		logger = newMemoryWriter()
	}

	log.SetOutput(&redirectGoLogger{})

	if *FlagLogSys && !IsRunningInteractive() {
		syslogLoggerCh = make(chan error)

		var err error

		syslogLogger, err = app.Service.Logger(syslogLoggerCh)
		Error(err)
	}

	if app != nil {
		prolog(fmt.Sprintf(">>> Start - %s %s %s", strings.ToUpper(app.Title), app.Version, strings.Repeat("-", 98)))
		prolog(fmt.Sprintf(">>> Cmdline : %s", strings.Join(SurroundWith(os.Args, "\""), " ")))
	}

	return nil
}

func logOutput(entry logEntry) {
	entryAsString := entry.toString(*FlagLogJson, *FlagLogVerbose)

	// fileLogger or memoryLogger
	if logger != nil {
		logger.writeString(entry.levelInt, fmt.Sprintf("%s\n", entryAsString))
	}

	if entry.levelInt == LEVEL_PROLOG {
		return
	}

	if syslogLogger != nil {
		msg := entry.Msg

		for i := 0; i < 2; i++ {
			msg = msg[strings.Index(msg, " ")+1:]
		}

		switch entry.levelInt {
		case LEVEL_WARN:
			Error(syslogLogger.Warning(msg))
		case LEVEL_ERROR:
			Error(syslogLogger.Error(msg))
		case LEVEL_PANIC:
			Error(syslogLogger.Error(msg))
		case LEVEL_DEBUG:
			fallthrough
		case LEVEL_INFO:
			Error(syslogLogger.Info(msg))
		}
	}

	if gotest != nil {
		switch entry.levelInt {
		case LEVEL_DEBUG:
			gotest.Logf(entryAsString)
		case LEVEL_INFO:
			gotest.Logf(entryAsString)
		default:
			gotest.Errorf(entryAsString)
		}
	} else {
		if entry.color != ColorDefault {
			entry.color.Println(entryAsString)
		} else {
			fmt.Println(entryAsString)
		}
	}
}

func closeLog() error {
	mu.Lock()
	defer mu.Unlock()

	isLogClosed = true

	close(logCh)

	wgLogCh.Wait()

	prolog(fmt.Sprintf("<<< End - %s %s %s", strings.ToUpper(app.Title), app.Version, strings.Repeat("-", 100)))

	if logger != nil {
		logger.close()
	}

	return nil
}

// logFile prints out the information
func prolog(t string, arg ...interface{}) {
	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	appendLog(LEVEL_PROLOG, ColorDefault, GetRuntimeInfo(1), t, nil)
}

func getRuntimePos(t string) (int, string) {
	pos := 1

	p := strings.Index(t, "~")
	if p != -1 {
		var err error

		pos, err = strconv.Atoi(t[:p])
		if err != nil {
			return 1, t
		}

		t = t[p+1:]
	}

	return pos, t
}

// Debug prints out the information
func Debug(t string, arg ...interface{}) {
	if !*FlagLogVerbose {
		return
	}

	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	riPos, t := getRuntimePos(t)

	appendLog(LEVEL_DEBUG, ColorDebug, GetRuntimeInfo(riPos), t, nil)
}

func loggingError(err error) bool {
	if err != nil && logger != nil {
		entry := newLogEntry(LEVEL_ERROR, ColorError, GetRuntimeInfo(1), err.Error())

		logger.writeString(LEVEL_DEBUG, fmt.Sprintf("%s\n", entry.toString(*FlagLogJson, true)))
	}

	return err != nil
}

// DebugError prints out the error
func DebugError(err error) bool {
	if !*FlagLogVerbose {
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
	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	riPos, t := getRuntimePos(t)

	appendLog(LEVEL_INFO, ColorInfo, GetRuntimeInfo(riPos), t, nil)
}

func Warn(t string, arg ...interface{}) {
	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	riPos, t := getRuntimePos(t)

	appendLog(LEVEL_WARN, ColorWarn, GetRuntimeInfo(riPos), t, nil)
}

func WarnError(err error) bool {
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
	if !*FlagLogVerbose {
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

func Panic(err error) {
	if err == nil || IsErrExit(err) {
		return
	}

	ri := GetRuntimeInfo(1)

	appendLog(LEVEL_PANIC, ColorPanic, ri, errorString(LEVEL_PANIC, ri, err), err)

	done()

	os.Exit(1)
}

func Error(err error) bool {
	if err != nil && !IsErrExit(err) && !IsSuppressedError(err) {
		ri := GetRuntimeInfo(1)

		appendLog(LEVEL_ERROR, ColorError, ri, errorString(LEVEL_ERROR, ri, err), err)

		if *FlagLogBreak {
			Panic(fmt.Errorf("BREAK ON ERROR"))
		}
	}

	return err != nil
}

func IsError(err error, target error) bool {
	return err != nil && reflect.TypeOf(err) == reflect.TypeOf(target)
}

func newLogEntry(level int, color color.Color, ri RuntimeInfo, msg string) logEntry {
	return logEntry{
		levelInt: level,
		color:    color,
		Level:    strings.ToUpper(levelToString(level)),
		Clock:    time.Now().Format(DateTimeMilliMask),
		Runtime:  ri.String(),
		Msg:      Capitalize(strings.TrimRight(strings.TrimSpace(msg), "\r\n")),
	}
}

func appendLog(level int, color color.Color, ri RuntimeInfo, msg string, err error) {
	mu.Lock()
	defer mu.Unlock()

	if isLogClosed {
		return
	}

	if level >= LEVEL_ERROR {
		if err.Error() == lastErr {
			return
		}

		lastErr = err.Error()
	} else {
		lastErr = ""
	}

	entry := newLogEntry(level, color, ri, msg)

	if gotest != nil {
		entryAsString := entry.toString(*FlagLogJson, *FlagLogVerbose)

		switch entry.levelInt {
		case LEVEL_DEBUG:
			gotest.Logf(entryAsString)
		case LEVEL_INFO:
			gotest.Logf(entryAsString)
		default:
			gotest.Errorf(entryAsString)
		}

		return
	}

	if level != LEVEL_PANIC && !*FlagLogVerbose {
		logCh <- entry
	} else {
		logOutput(entry)
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

	return logger.getLogs(w)
}

func ClearLogs() error {
	if logger == nil {
		return fmt.Errorf("no logger")
	}

	return logger.clearLogs()
}

func LogsAvailable() bool {
	return logger != nil
}

func TraceError(err error) error {
	Error(err)

	return err
}
