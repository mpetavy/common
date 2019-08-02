package common

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
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

var (
	logLevel       = LEVEL_INFO
	logConsole     *bool
	logFilename    *string
	logFileSize    *int
	logLevelString *string
	logFileBackup  *int

	defaultLogFile string
	logEntries     chan logEntry
	mutex          sync.Mutex
	logFile        *os.File
)

func init() {
	defaultLogFile = CustomAppFilename(".log")

	logConsole = flag.Bool("logconsole", true, "log to console")
	logFilename = flag.String("logfile", "", fmt.Sprintf("filename to log logFile (use \".\" for %s)", defaultLogFile))
	logFileSize = flag.Int("logfilesize", 10, "log logFile size in MB")
	logFileBackup = flag.Int("logfilebackup", 5, "logFile backups")
	logLevelString = flag.String("loglevel", "info", "log level (debug,info,error,fatal)")
}

func initLog() {
	if logEntries == nil {
		if logEntries == nil {
			AddShutdownHook(func() error {
				return closeLogfile()
			})

			logEntries = make(chan logEntry, 100)

			switch strings.ToLower(*logLevelString) {
			case "debug":
				logLevel = LEVEL_DEBUG
			case "info":
				logLevel = LEVEL_INFO
			case "warn":
				logLevel = LEVEL_WARN
			case "error":
				logLevel = LEVEL_ERROR
			case "fatal":
				logLevel = LEVEL_FATAL
			default:
				logLevel = LEVEL_INFO
			}

			if *logFilename == "." {
				*logFilename = defaultLogFile
			}

			go func() {
				for entry := range logEntries {
					writeEntry(entry)
				}
			}()

			if app != nil {
				prolog(fmt.Sprintf(">>> START - %s %s", strings.ToUpper(app.Name), app.Version))
				prolog(fmt.Sprintf("cmdline : %s", strings.Join(SurroundWith(os.Args, "\""), " ")))
			}
		}
	}
}

func writeEntry(entry logEntry) {
	mutex.Lock()
	defer mutex.Unlock()

	var err error

	if logFile == nil && len(*logFilename) != 0 {
		b, _ := FileExists(*logFilename)

		if b {
			fi, _ := os.Stat(*logFilename)

			if fi.Size() > (int64(*logFileSize) * 1024 * 1024) {
				err := FileBackup(*logFilename, *logFileBackup)
				Fatal(fmt.Errorf("cannot write to logFile %s: %v", *logFilename, err))
			}
		}

		logFile, err = os.OpenFile(*logFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)

		if err != nil {
			Error(err)

			return
		}
	}

	if entry.level != LEVEL_FILE && *logConsole {
		fmt.Fprintf(os.Stderr, "%s\n", entry.String())
	}

	if logFile != nil {
		_, err = logFile.WriteString(fmt.Sprintf("%s\n", entry.String()))
		DebugError(logFile.Sync())
	}

	if err != nil {
		Error(err)
	}
}

func closeLogfile() error {
	mutex.Lock()
	defer mutex.Unlock()

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

	log(LEVEL_DEBUG, RuntimeInfo(2), t)
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
		t = fmt.Sprintf("%s: %v", t, arg[0])
	}
	if len(arg) > 1 {
		t = fmt.Sprintf("%s: "+fmt.Sprintf("%v", arg[0]), arg[1:])
	}

	log(LEVEL_DEBUG, ri, t)
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
		log(LEVEL_FATAL, RuntimeInfo(1), err.Error())

		panic(err)
	}
}

func log(level int, ri runtimeInfo, msg string) {
	if logEntries != nil && (level == LEVEL_FILE || level >= logLevel) {
		logEntries <- logEntry{
			level: level,
			ri:    ri,
			msg:   msg,
		}
	}
}

func ToString(cmd exec.Cmd) string {
	s := SurroundWith(cmd.Args, "\"")

	return strings.Join(s, " ")
}

func IsDebugMode() bool {
	initLog()

	return logLevel == LEVEL_DEBUG
}
