package common

import (
	"flag"
	"fmt"
	"github.com/kardianos/service"
	"os"
	"path/filepath"
	"strings"
	"time"

	"os/exec"
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

type ErrExit struct {
}

func (e *ErrExit) Error() string { return "" }

var (
	LogEnabled = NewSignal()
)

type logEntry struct {
	level int
	ri    runtimeInfo
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
	return strings.TrimRight(fmt.Sprintf("%s %-5s %-40.40s %s", time.Now().Format(DateTimeMilliMask), level, l.ri.String(false), Capitalize(l.msg)), "\r\n")
}

type DebugWriter struct {
	Name   string
	Action string
}

func (this *DebugWriter) Write(p []byte) (n int, err error) {
	Debug("%s %s %d bytes: %+q", this.Name, this.Action, len(p), string(p))

	return len(p), nil
}

var (
	logLevel    *string
	logFilename *string
	logFileSize *int

	defaultLogFile string
	logFile        *os.File
	signCount      = NewSignal()
)

func init() {
	path := CleanPath(AppFilename(".log"))
	if !IsWindowsOS() && !service.Interactive() {
		path = filepath.Join("var", "log", AppFilename(".log"))
	}

	defaultLogFile = path

	logFilename = flag.String("logfile", "", fmt.Sprintf("filename to log logFile (use \".\" for %s)", defaultLogFile))
	logFileSize = flag.Int("logfilesize", 1048576, "log logFile size in bytes")
	logLevel = flag.String("loglevel", "info", "log level (debug,info,error,fatal)")
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

	closeLogfile(false)

	if *logFilename == "." {
		*logFilename = defaultLogFile
	}

	openLogFile()

	LogEnabled.Set()

	if app != nil {
		prolog(fmt.Sprintf(">>> Start - %s %s %s", strings.ToUpper(app.Name), app.Version, strings.Repeat("-", 98)))
		prolog(fmt.Sprintf(">>> Cmdline : %s", strings.Join(SurroundWith(os.Args, "\""), " ")))
	}
}

func writeEntry(entry logEntry) {
	if !LogEnabled.IsSet() {
		return
	}

	if entry.level != LEVEL_FILE {
		s := entry.String()
		if currentLevel() > LEVEL_DEBUG && len(s) > 71 {
			s = s[71:]
		}

		_, err := fmt.Printf("%s\n", s)
		IgnoreError(err)
	}

	if logFile != nil {
		if signCount.IncAndReached(100) {
			closeLogfile(false)
			openLogFile()

			signCount.ResetWithoutLock()
			signCount.Unlock()
		}

		// dont handle errors here
		_, err := logFile.WriteString(fmt.Sprintf("%s\n", entry.String()))
		IgnoreError(err)

		// dont handle errors here
		IgnoreError(logFile.Sync())
	}
}

func openLogFile() {
	if *logFilename != "" && logFile == nil {
		b, _ := FileExists(*logFilename)

		if b {
			fi, _ := os.Stat(*logFilename)

			if fi.Size() > int64(*logFileSize) {
				Error(FileBackup(*logFilename))
			}
		}

		var err error

		logFile, err = os.OpenFile(*logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, FileMode(true, true, false))

		if err != nil {
			logFile = nil
		}
	}
}

func closeLogfile(final bool) {
	if final {
		prolog(fmt.Sprintf("<<< End - %s %s %s", strings.ToUpper(app.Name), app.Version, strings.Repeat("-", 100)))
	}

	if logFile != nil {
		// dont handle errors here
		IgnoreError(logFile.Close())

		logFile = nil
	}
}

// logFile prints out the information
func prolog(t string, arg ...interface{}) {
	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_FILE, RuntimeInfo(1), t)
}

// Debug prints out the information
func Debug(t string, arg ...interface{}) {
	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_DEBUG, RuntimeInfo(1), t)
}

// Info prints out the information
func Info(t string, arg ...interface{}) {
	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_INFO, RuntimeInfo(1), t)
}

// Warn prints out the information
func Warn(t string, arg ...interface{}) {
	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_WARN, RuntimeInfo(1), t)
}

// Warn prints out the error
func WarnError(err error) {
	if err != nil {
		log(LEVEL_WARN, RuntimeInfo(1), err.Error())
	}
}

// DebugFunc prints out the current executon func
func DebugFunc(arg ...interface{}) {
	ri := RuntimeInfo(1)

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

// IgnoreError just ignores the error
func IgnoreError(_ error) {
}

// Debug prints out the information
func DebugError(err error) {
	if err != nil {
		log(LEVEL_DEBUG, RuntimeInfo(1), fmt.Sprintf("DebugError: %s", err.Error()))
	}
}

// Error prints out the error
func Error(err error) {
	if err != nil {
		log(LEVEL_ERROR, RuntimeInfo(1), err.Error())
	}
}

// Fatal prints out the error
func Fatal(err error) {
	if err != nil {

		if _, ok := err.(*ErrExit); !ok {
			log(LEVEL_FATAL, RuntimeInfo(1), err.Error())

			panic(err)
		}
	}
}

func log(level int, ri runtimeInfo, msg string) {
	if !LogEnabled.IsSet() {
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

func CheckError(err error) bool {
	b := err != nil

	if b {
		Error(err)
	}

	return b
}

func LogFileName() string {
	return *logFilename
}
