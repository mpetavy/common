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
	txt   string
}

var (
	level          = LEVEL_INFO
	logConsole     *bool
	logFile        *string
	logFileSize    *int
	logLevel       *string
	defaultLogFile string
	logEntries     chan logEntry
	mutex          sync.Mutex
	wg             sync.WaitGroup
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
	logFile = flag.String("logfile", "", fmt.Sprintf("filename to log file (use \".\" for %s)", defaultLogFile))
	logFileSize = flag.Int("logfilesize", 10, "log file size in MB")
	logLevel = flag.String("loglevel", "info", "log level (debug,info,error,fatal)")
}

func initLog() {
	var err error

	logEntries = make(chan logEntry, 100)

	switch strings.ToLower(*logLevel) {
	case "debug":
		level = LEVEL_DEBUG
	case "info":
		level = LEVEL_INFO
	case "warn":
		level = LEVEL_WARN
	case "error":
		level = LEVEL_ERROR
	case "fatal":
		level = LEVEL_FATAL
	default:
		level = LEVEL_INFO
	}

	if *logFile == "." {
		*logFile = defaultLogFile
	}

	File(fmt.Sprintf(">>> START - %s %s", strings.ToUpper(app.Name), app.Version))
	File(fmt.Sprintf("cmdline : %s", strings.Join(SurroundWith(os.Args, "\""), " ")))

	go func() {
		var file *os.File

		timeout := time.Millisecond * 500

		t := time.NewTimer(timeout)

	loop:
		for {
			select {
			case <-t.C:
				mutex.Lock()

				t.Stop()

				if file != nil {
					file.Close()

					fi, _ := os.Stat(*logFile)

					if fi.Size() > (int64(*logFileSize) * 1024 * 1024) {
						os.Remove(*logFile)
					}

					file = nil
				}

				mutex.Unlock()
			case entry, ok := <-logEntries:
				if !ok {
					break loop
				}

				mutex.Lock()

				t.Stop()

				entry.txt = strings.TrimRight(entry.txt, "\r\n")

				if file == nil && len(*logFile) != 0 {
					file, err = os.OpenFile(*logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
					if err != nil {
						Fatal(fmt.Errorf("cannot write to file %s: %v", *logFile, err))
					}
				}

				if entry.level != LEVEL_FILE && *logConsole {
					fmt.Printf("%s\n", entry.txt)
				}

				if file != nil {
					file.WriteString(fmt.Sprintf("%s\n", entry.txt))
				}

				mutex.Unlock()

				t.Reset(timeout)

				wg.Done()
			}
		}

		mutex.Lock()

		if file != nil {
			file.Close()

			file = nil
		}

		mutex.Unlock()
	}()

	AddShutdownHook(func() error {
		return closeLogFile()
	})
}

func closeLogFile() error {
	DebugFunc()

	if logEntries != nil {
		File(fmt.Sprintf("<<< STOP - %s %s", strings.ToUpper(app.Name), app.Version))

		close(logEntries)

		wg.Wait()
	}

	return nil
}

func fmtLog(level string, pos int, txt string) string {
	ri := RuntimeInfo(pos + 1)

	return fmt.Sprintf("%s %-5s %-40.40s %s", time.Now().Format(DateTimeMilliMask), level, ri.String(false), Capitalize(txt))
}

// File prints out the information
func File(txt string) {
	log(LEVEL_FILE, fmtLog("FILE ", 2, txt))
}

// Info prints out the information
func Info(t string, arg ...interface{}) {
	if level <= LEVEL_INFO {
		if len(arg) > 0 {
			t = fmt.Sprintf(t, arg...)
		}

		log(LEVEL_INFO, fmtLog("INFO ", 2, t))
	}
}

// Warn prints out the error
func WarnError(err error) {
	if err != nil {
		if level <= LEVEL_ERROR {
			log(LEVEL_WARN, fmtLog("WARN", 2, err.Error()))
		}
	}
}

// Warn prints out the information
func Warn(t string, arg ...interface{}) {
	if level <= LEVEL_WARN {
		if len(arg) > 0 {
			t = fmt.Sprintf(t, arg...)
		}

		log(LEVEL_WARN, fmtLog("WARN ", 2, t))
	}
}

// Debug prints out the information
func Debug(t string, arg ...interface{}) {
	if level <= LEVEL_DEBUG {
		if len(arg) > 0 {
			t = fmt.Sprintf(t, arg...)
		}

		log(LEVEL_DEBUG, fmtLog("DEBUG", 2, t))
	}
}

// DebugFunc prints out the current executon func
func DebugFunc() {
	ri := RuntimeInfo(1)

	if level <= LEVEL_DEBUG {
		log(LEVEL_DEBUG, fmtLog("DEBUG", 2, ri.Fn+"()"))
	}
}

// Debug prints out the information
func DebugError(err error) {
	if level <= LEVEL_DEBUG {
		log(LEVEL_DEBUG, fmtLog("DEBUG", 2, fmt.Sprintf("Error: %s", err.Error())))
	}
}

// Error prints out the error
func Error(err error) {
	if err != nil {
		if level <= LEVEL_ERROR {
			log(LEVEL_ERROR, fmtLog("ERROR", 2, err.Error()))
		}
	}
}

// Fatal prints out the error
func Fatal(err error) {
	if err != nil {
		s := fmtLog("FATAL", 2, err.Error())
		if level <= LEVEL_FATAL {
			log(LEVEL_FATAL, s)
		}

		panic(err)
	}
}

func log(level int, txt string) {
	mutex.Lock()

	wg.Add(1)

	logEntries <- logEntry{level, txt}

	mutex.Unlock()
}

func ToString(cmd exec.Cmd) string {
	s := SurroundWith(cmd.Args, "\"")

	return strings.Join(s, " ")
}

func IsDebugMode() bool {
	return level == LEVEL_DEBUG
}
