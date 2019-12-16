package common

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gookit/color"
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
	FlagLogVerbose     *bool
	logFilename        *string
	logFilesize        *int64
	logJson            *bool
	logger             logWriter
	defaultLogFilename string
	mu                 sync.Mutex
)

func init() {
	defaultLogFilename = CleanPath(AppFilename(".log"))

	logFilename = flag.String("log.file", "", fmt.Sprintf("filename to log logFile (use \".\" for %s)", defaultLogFilename))
	logFilesize = flag.Int64("log.filesize", 5*1024*1024, "max log file size")
	FlagLogVerbose = flag.Bool("log.verbose", false, "verbose logging")
	logJson = flag.Bool("log.json", false, "JSON output")
}

type ErrExit struct {
}

func (e *ErrExit) Error() string { return "" }

type logEntry struct {
	levelInt int
	Clock    string `json:"time"`
	LevelStr string `json:"level"`
	Ri       string `json:"runtime"`
	Msg      string `json:"msg"`
}

func (l *logEntry) String() string {
	if *logJson {
		ba, _ := json.Marshal(l)
		return string(ba)

	} else {
		if FlagLogVerbose == nil || *FlagLogVerbose {
			return fmt.Sprintf("%s %s %-40.40s %s", l.Clock, FillString(l.LevelStr+":", 6, false, " "), l.Ri, l.Msg)
		} else {
			return fmt.Sprintf("%s %s", FillString(l.LevelStr+":", 6, false, " "), l.Msg)
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
		lines: make([]string, 1000),
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

	if this.filesize >= *logFilesize {
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
	for i := *countBackups; i >= 0; i-- {
		var src string

		if *countBackups == 1 {
			src = realLogFilename() + ".bak"

			b, _ := fileExists(src)
			if !b {
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

func newLogFileWriter() *logFileWriter {
	filesize, _ := FileSize(realLogFilename())
	logFile, _ := os.OpenFile(realLogFilename(), os.O_RDWR|os.O_CREATE|os.O_APPEND, DefaultFileMode)

	writer := logFileWriter{
		file:     logFile,
		filesize: filesize,
	}

	return &writer
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
	case LEVEL_FATAL:
		return "FATAL"
	default:
		return "INFO"
	}
}

func initLog() {
	DebugFunc()

	if realLogFilename() != "" {
		if realLogFilename() == "memory" {
			logger = newLogMemoryWriter()
		} else {
			logger = newLogFileWriter()
		}
	}

	if app != nil {
		prolog(fmt.Sprintf(">>> Start - %s %s %s", strings.ToUpper(app.Name), app.Version, strings.Repeat("-", 98)))
		prolog(fmt.Sprintf(">>> Cmdline : %s", strings.Join(SurroundWith(os.Args, "\""), " ")))
	}
}

func writeEntry(entry logEntry) {
	if entry.levelInt != LEVEL_FILE {
		s := entry.String()

		switch entry.levelInt {
		case LEVEL_WARN:
			color.Warn.Println(s)
		case LEVEL_ERROR:
			color.Error.Println(s)
		case LEVEL_FATAL:
			color.Error.Println(s)
		default:
			fmt.Printf("%s\n", s)
		}
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
	return fmt.Sprintf("%s [%T]", err.Error(), err)
}

// Warn prints out the error
func WarnError(err error) bool {
	if err != nil && !isErrExit(err) {
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

// Error prints out the error
func Error(err error) bool {
	if err != nil && !isErrExit(err) {
		log(LEVEL_ERROR, GetRuntimeInfo(1), errorString(err))
	}

	return err != nil
}

func isErrExit(err error) bool {
	if err == nil {
		return false
	}

	_, ok := err.(*ErrExit)
	return ok
}

// Fatal prints out the error
func Fatal(err error) bool {
	if err != nil && !isErrExit(err) {
		log(LEVEL_FATAL, GetRuntimeInfo(1), errorString(err))

		panic(err)
	}

	return err != nil
}

func log(level int, ri RuntimeInfo, msg string) {
	mu.Lock()
	defer mu.Unlock()

	if level == LEVEL_FILE || (FlagLogVerbose != nil && *FlagLogVerbose) || level > LEVEL_DEBUG {
		writeEntry(logEntry{
			levelInt: level,
			LevelStr: levelToString(level),
			Clock:    time.Now().Format(DateTimeMilliMask),
			Ri:       ri.String(),
			Msg:      Capitalize(strings.TrimRight(strings.TrimSpace(msg), "\r\n")),
		})
	}
}

func ToString(cmd exec.Cmd) string {
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
	if *logFilename == "." {
		return defaultLogFilename
	} else {
		return *logFilename
	}
}
