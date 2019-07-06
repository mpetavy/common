package common

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"os/exec"
)

const (
	// DB level
	LEVEL_PROLOG = iota
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
	txt   string
}

type ReentrantLock struct {
	locked bool
	sync.Mutex
}

func (rl *ReentrantLock) Lock() {
	if !rl.locked {
		rl.locked = true

		rl.Lock()
	}
}

func (rl *ReentrantLock) Unlock() {
	if rl.locked {
		rl.locked = false

		rl.Unlock()
	}
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
	mutex          ReentrantLock
	logFile        *os.File
)

func init() {
	filename, err := os.Executable()
	if err != nil {
		filename = os.Args[0]
	}

	ext := filepath.Ext(filename)

	if len(ext) > 0 {
		filename = string(filename[:len(filename)-len(ext)])
	}

	filename += ".log"

	path, err := os.Getwd()
	if err == nil {
		filename = filepath.Join(path, filepath.Base(filename))
	}

	defaultLogFile = filename

	logConsole = flag.Bool("logconsole", true, "log to console")
	logFilename = flag.String("logfile", "", fmt.Sprintf("filename to log logFile (use \".\" for %s)", defaultLogFile))
	logFileSize = flag.Int("logfilesize", 10, "log logFile size in MB")
	logFileBackup = flag.Int("logfilebackup", 5, "logFile backups")
	logLevelString = flag.String("loglevel", "info", "log level (debug,info,error,fatal)")
}

func initLog() {
	if logEntries == nil {
		mutex.Lock()
		defer mutex.Unlock()

		if logEntries == nil {
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
				timeout := time.Millisecond * 500

				t := time.NewTimer(timeout)

			loop:
				for {
					select {
					case <-t.C:
						t.Stop()

						closeLogFile(false)
					case entry, ok := <-logEntries:
						t.Stop()

						if !ok {
							mutex.Unlock()

							break loop
						}

						mutex.Lock()

						writeEntry(entry)

						t.Reset(timeout)
					}
				}

				closeLogFile(true)
			}()

			if app != nil {
				prolog(fmt.Sprintf(">>> START - %s %s", strings.ToUpper(app.Name), app.Version))
				prolog(fmt.Sprintf("cmdline : %s", strings.Join(SurroundWith(os.Args, "\""), " ")))
			}
		}
	}
}

func writeEntry(entry logEntry) {
	//mutex.Lock()
	//mutex.Unlock()

	entry.txt = strings.TrimRight(entry.txt, "\r\n")

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

	if entry.level != LEVEL_PROLOG && *logConsole {
		fmt.Fprintf(os.Stderr, "%s\n", entry.txt)
	}

	if logFile != nil {
		_, err = logFile.WriteString(fmt.Sprintf("%s\n", entry.txt))
	}

	if err != nil {
		Error(err)
	}
}

func closeLogFile(isFinal bool) {
	mutex.Lock()
	mutex.Unlock()

	if isFinal {
		if app != nil {
			prolog(fmt.Sprintf("<<< STOP - %s %s", strings.ToUpper(app.Name), app.Version))
		}
	}

	if isFinal {
		close(logEntries)

		logEntries = nil
	}

	if logFile != nil {
		logFile.Close()

		logFile = nil
	}
}

func fmtLog(level string, pos int, txt string) string {
	ri := RuntimeInfo(pos + 1)

	return fmt.Sprintf("%s %-5s %-40.40s %s", time.Now().Format(DateTimeMilliMask), level, ri.String(false), Capitalize(txt))
}

// logFile prints out the information
func prolog(t string, arg ...interface{}) {
	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_PROLOG, fmtLog("FILE", 2, t))
}

// Debug prints out the information
func Debug(t string, arg ...interface{}) {
	initLog()

	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_DEBUG, fmtLog("DEBUG", 2, t))
}

// Info prints out the information
func Info(t string, arg ...interface{}) {
	initLog()

	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_INFO, fmtLog("INFO", 2, t))
}

// Warn prints out the information
func Warn(t string, arg ...interface{}) {
	initLog()

	if len(arg) > 0 {
		t = fmt.Sprintf(t, arg...)
	}

	log(LEVEL_WARN, fmtLog("WARN", 2, t))
}

// Warn prints out the error
func WarnError(err error) {
	initLog()

	if err != nil {
		log(LEVEL_WARN, fmtLog("WARN", 2, err.Error()))
	}
}

// DebugFunc prints out the current executon func
func DebugFunc() {
	initLog()

	ri := RuntimeInfo(1)

	log(LEVEL_DEBUG, fmtLog("DEBUG", 2, ri.Fn+"()"))
}

// Debug prints out the information
func DebugError(err error) {
	initLog()

	if err != nil {
		log(LEVEL_DEBUG, fmtLog("DEBUG", 2, err.Error()))
	}
}

// Error prints out the error
func Error(err error) {
	initLog()

	if err != nil {
		log(LEVEL_ERROR, fmtLog("ERROR", 2, err.Error()))
	}
}

// Fatal prints out the error
func Fatal(err error) {
	initLog()

	if err != nil {
		log(LEVEL_FATAL, fmtLog("FATAL", 2, err.Error()))

		panic(err)
	}
}

func log(level int, txt string) {
	if logEntries != nil && (level == LEVEL_PROLOG || level >= logLevel) {
		logEntries <- logEntry{level, txt}
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
