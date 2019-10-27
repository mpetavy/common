package common

import (
	"flag"
	"fmt"
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
	// LEVEL_FATAL level
	LEVEL_FATAL
)

var (
	logLevel           *string
	logFilename        *string
	logSize            *int
	logEnabled         = NewNotice()
	logger             logWriter
	defaultLogFilename string
)

func init() {
	defaultLogFilename = CleanPath(AppFilename(".log"))

	logFilename = flag.String("log.file", "", fmt.Sprintf("filename to log logFile (use \".\" for %s)", defaultLogFilename))
	logSize = flag.Int("log.size", 1000, "max amount of log lines")
	logLevel = flag.String("log.level", "info", "log level (debug,info,error,fatal)")
}

type ErrExit struct {
}

func (e *ErrExit) Error() string { return "" }

type logEntry struct {
	level int
	ri    RuntimeInfo
	msg   string
}

func (l *logEntry) String() string {
	level := ""

	switch l.level {
	case LEVEL_FILE:
		level = "FILE"
	case LEVEL_DEBUG:
		level = "DEBUG"
	case LEVEL_INFO:
		level = "INFO"
	case LEVEL_WARN:
		level = "WARN"
	case LEVEL_ERROR:
		level = "ERROR"
	case LEVEL_FATAL:
		level = "FATAL"
	}
	return strings.TrimRight(fmt.Sprintf("%s %-5s %-40.40s %s", time.Now().Format(DateTimeMilliMask), level, l.ri.String(), Capitalize(l.msg)), "\r\n")
}

type logWriter interface {
	WriteString(txt string)
	Logs(io.Writer) error
	Close()
}

type logMemoryWriter struct {
	mu    sync.Mutex
	lines []string
}

func (this *logMemoryWriter) WriteString(txt string) {
	this.mu.Lock()
	defer this.mu.Unlock()

	copy(this.lines[0:], this.lines[1:])
	this.lines[len(this.lines)-1] = txt
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
		mu:    sync.Mutex{},
		lines: make([]string, *logSize),
	}

	return &writer
}

type logFileWriter struct {
	c    int
	mu   sync.Mutex
	file *os.File
}

func (this *logFileWriter) WriteString(txt string) {
	this.mu.Lock()
	defer this.mu.Unlock()

	logEnabled.Unset()
	defer logEnabled.Set()

	if this.file == nil {
		return
	}

	if this.c == *logSize {
		this.c = 0

		if this.file != nil {
			Ignore(this.file.Close())
			this.file = nil
		}

		Ignore(FileBackup(*logFilename))

		this.file, _ = os.OpenFile(*logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, FileMode(true, true, false))
	}

	if this.file == nil {
		return
	}

	Ignore(this.file.Write([]byte(txt)))
	Ignore(this.file.Sync())

	this.c++
}

func (this *logFileWriter) Logs(w io.Writer) error {
	for i := *countBackups; i >= 0; i-- {
		var src string

		if *countBackups == 1 {
			src = *logFilename + ".bak"
		} else {
			if i > 0 {
				src = *logFilename + "." + strconv.Itoa(i)
			} else {
				src = *logFilename
			}
		}

		b, _ := FileExists(src)

		if b {
			file, err := os.Open(src)
			if err != nil {
				return err
			}

			_, err = io.Copy(w, file)
			Ignore(file.Close())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (this *logFileWriter) Close() {
	this.mu.Lock()
	defer this.mu.Unlock()

	if this.file != nil {
		Ignore(this.file.Close())
		this.file = nil
	}
}

func newLogFileWriter() *logFileWriter {
	logFile, _ := os.OpenFile(*logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, FileMode(true, true, false))

	writer := logFileWriter{
		mu:   sync.Mutex{},
		file: logFile,
	}

	return &writer
}

func currentLevel() int {
	switch strings.ToLower(*logLevel) {
	case "debug":
		return LEVEL_DEBUG
	case "info":
		return LEVEL_INFO
	case "warn":
		return LEVEL_WARN
	case "error":
		return LEVEL_ERROR
	case "fatal":
		return LEVEL_FATAL
	default:
		return LEVEL_INFO
	}
}

func initLog() {
	DebugFunc()

	if *logFilename == "." {
		*logFilename = defaultLogFilename
	}

	if *logFilename != "" {
		if *logFilename == "memory" {
			logger = newLogMemoryWriter()
		} else {
			logger = newLogFileWriter()
		}
	}

	logEnabled.Set()

	if app != nil {
		prolog(fmt.Sprintf(">>> Start - %s %s %s", strings.ToUpper(app.Name), app.Version, strings.Repeat("-", 98)))
		prolog(fmt.Sprintf(">>> Cmdline : %s", strings.Join(SurroundWith(os.Args, "\""), " ")))
	}
}

func writeEntry(entry logEntry) {
	if !logEnabled.IsSet() {
		return
	}

	if entry.level != LEVEL_FILE {
		s := entry.String()
		if currentLevel() > LEVEL_DEBUG {
			if entry.level > LEVEL_WARN && len(s) > 71 {
				s = s[24:]
			} else {
				s = s[71:]
			}
		}
		fmt.Printf("%s\n", s)
	}

	if logger != nil {
		logger.WriteString(fmt.Sprintf("%s\n", entry.String()))
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

	log(LEVEL_FILE, GetRuntimeInfo(1), t)
}

// Debug prints out the information
func Debug(t string, arg ...interface{}) {
	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_DEBUG, GetRuntimeInfo(1), t)
}

// Info prints out the information
func Info(t string, arg ...interface{}) {
	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_INFO, GetRuntimeInfo(1), t)
}

// Warn prints out the information
func Warn(t string, arg ...interface{}) {
	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_WARN, GetRuntimeInfo(1), t)
}

func errorString(err error) string {
	return fmt.Sprintf("%T: %s", err, err.Error())
}

// Warn prints out the error
func WarnError(err error) bool {
	if err != nil {
		log(LEVEL_WARN, GetRuntimeInfo(1), errorString(err))
	}

	return err != nil
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

	log(LEVEL_DEBUG, ri, t)
}

// Ignore just ignores the error
func Ignore(arg ...interface{}) bool {
	b := false

	if len(arg) > 0 {
		_, b = arg[len(arg)-1].(error)
	}

	return b
}

// Debug prints out the information
func DebugError(err error) bool {
	if err != nil {
		log(LEVEL_DEBUG, GetRuntimeInfo(1), fmt.Sprintf("DebugError: %s", errorString(err)))
	}

	return err != nil
}

// Error prints out the error
func Error(err error) bool {
	if err != nil {
		log(LEVEL_ERROR, GetRuntimeInfo(1), errorString(err))
	}

	return err != nil
}

// Fatal prints out the error
func Fatal(err error) bool {
	if err != nil {
		if _, ok := err.(*ErrExit); !ok {
			log(LEVEL_FATAL, GetRuntimeInfo(1), errorString(err))

			panic(err)
		}
	}

	return err != nil
}

func log(level int, ri RuntimeInfo, msg string) {
	if !logEnabled.IsSet() {
		return
	}

	if level == LEVEL_FILE || level >= currentLevel() {
		writeEntry(logEntry{
			level: level,
			ri:    ri,
			msg:   msg,
		})
	}
}

func ToString(cmd exec.Cmd) string {
	s := SurroundWith(cmd.Args, "\"")

	return strings.Join(s, " ")
}

func Logs(w io.Writer) error {
	if logger != nil {
		return logger.Logs(w)
	}
	return nil
}
