package common

import (
	"flag"
	"fmt"
	"os"
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
	logLevel      *string
	logFilename   *string
	logFileSize   *int
	logFileBackup *int

	defaultLogFile string
	logFile        *os.File
	signInitLog    Sign
	signCount      Sign
)

func init() {
	defaultLogFile = AppFilename(".log")

	logFilename = flag.String("logfile", "", fmt.Sprintf("filename to log logFile (use \".\" for %s)", defaultLogFile))
	logFileSize = flag.Int("logfilesize", 1048576, "log logFile size in bytes")
	logFileBackup = flag.Int("logfilebackup", 5, "logFile backups")
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
	if signInitLog.Set() {
		if *logFilename == "." {
			*logFilename = defaultLogFile
		}

		openLogFile()

		if app != nil {
			prolog(fmt.Sprintf(">>> Start - %s %s %s", strings.ToUpper(app.Name), app.Version, strings.Repeat("-", 100)))
			prolog(fmt.Sprintf(">>> Cmdline : %s", strings.Join(SurroundWith(os.Args, "\""), " ")))
		}
	}
}

func writeEntry(entry logEntry) {
	if entry.level != LEVEL_FILE {
		fmt.Fprintf(os.Stderr, "%s\n", entry.String())
	}

	if logFile != nil {
		if signCount.IncAndReached(100) {
			closeLogfile(false)
			openLogFile()

			signCount.ResetWithoutLock()
			signCount.Unlock()
		}

		logFile.WriteString(fmt.Sprintf("%s\n", entry.String()))
		logFile.Sync()
	}
}

func openLogFile() {
	if *logFilename != "" {
		b, _ := FileExists(*logFilename)

		if b {
			fi, _ := os.Stat(*logFilename)

			if fi.Size() > int64(*logFileSize) {
				err := FileBackup(*logFilename, *logFileBackup)
				Fatal(fmt.Errorf("cannot backup logFile %s: %v", *logFilename, err))
			}
		}

		var err error

		logFile, err = os.OpenFile(*logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)

		if err != nil {
			logFile.Close()

			logFile = nil

			Error(err)
		}
	}
}

func closeLogfile(final bool) error {
	if final {
		prolog(fmt.Sprintf("<<< End - %s %s %s", strings.ToUpper(app.Name), app.Version, strings.Repeat("-", 100)))
	}

	if logFile != nil {
		logFile.Close()

		logFile = nil
	}

	return nil
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
	initLog()

	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_DEBUG, RuntimeInfo(1), t)
}

// Info prints out the information
func Info(t string, arg ...interface{}) {
	initLog()

	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_INFO, RuntimeInfo(1), t)
}

// Warn prints out the information
func Warn(t string, arg ...interface{}) {
	initLog()

	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_WARN, RuntimeInfo(1), t)
}

// Warn prints out the error
func WarnError(err error) {
	initLog()

	if err != nil {
		log(LEVEL_WARN, RuntimeInfo(1), err.Error())
	}
}

// DebugFunc prints out the current executon func
func DebugFunc(arg ...interface{}) {
	initLog()

	ri := RuntimeInfo(1)

	t := ri.Fn + "()"

	if len(arg) == 1 {
		t = fmt.Sprintf("%s: %s", t, arg[0])
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
func IgnoreError(err error) {
}

// Debug prints out the information
func DebugError(err error) {
	initLog()

	if err != nil {
		log(LEVEL_DEBUG, RuntimeInfo(1), err.Error())
	}
}

// Error prints out the error
func Error(err error) {
	initLog()

	if err != nil {
		log(LEVEL_ERROR, RuntimeInfo(1), err.Error())
	}
}

// Fatal prints out the error
func Fatal(err error) {
	initLog()

	if err != nil {

		if _, ok := err.(*ErrExit); !ok {
			log(LEVEL_FATAL, RuntimeInfo(1), err.Error())

			panic(err)
		}
	}
}

func log(level int, ri runtimeInfo, msg string) {
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

func AppsInfo() *App {
	return app
}
