package common

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
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
	FlagNameLogGap      = "log.gap"
)

const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
	LevelFatal = "FATAL"
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
	FlagLogGap      = flag.Int(FlagNameLogGap, 100, "time gap after show a separator")

	mu        ReentrantMutex
	fw        *fileWriter
	rw                    = newMemoryWriter()
	LogDebug  *log.Logger = log.New(rw, prefix(LevelDebug), 0)
	LogInfo   *log.Logger = log.New(rw, prefix(LevelInfo), 0)
	LogWarn   *log.Logger = log.New(rw, prefix(LevelWarn), 0)
	LogError  *log.Logger = log.New(os.Stderr, prefix(LevelError), 0)
	LogFatal  *log.Logger = log.New(os.Stderr, prefix(LevelFatal), 0)
	lastErr   string
	lastLog   = time.Now()
	isLogInit bool
	tt        testingT
	testT     = NewSyncOf(tt)
)

func init() {
	Events.AddListener(EventFlagsParsed{}, func(event Event) {
		if *FlagLogSys && IsLinux() && !IsRunningInteractive() {
			// with SYSTEMD everything which is printed to console is automatically printed to journalctl

			*FlagLogSys = false
		}
	})
}

func IsLogVerboseEnabled() bool {
	if FlagLogVerbose != nil {
		return *FlagLogVerbose
	}

	for _, arg := range os.Args {
		if arg == "-"+FlagNameLogVerbose || arg == "-"+FlagNameLogVerbose+"=true" {
			return true
		}
	}

	return false
}

func IsLogInit() bool {
	return isLogInit
}

func IsLogJsonEnabled() bool {
	if FlagLogJson != nil {
		return *FlagLogJson
	}

	for _, arg := range os.Args {
		if arg == "-"+FlagNameLogJson || arg == "-"+FlagNameLogJson+"=true" {
			return true
		}
	}

	return false
}

func IsLogFileEnabled() bool {
	if FlagLogFileName != nil {
		return *FlagLogFileName != ""
	}

	for _, arg := range os.Args {
		if arg == "-"+FlagNameLogFileName || strings.HasPrefix(arg, FlagNameLogFileName+"=") {
			return true
		}
	}

	return false
}

func prefix(p string) string {
	if IsLogVerboseEnabled() && !IsLogJsonEnabled() {
		return fmt.Sprintf("%-6s", p)
	}

	return ""
}

func initLog() error {
	mu.Lock()
	defer mu.Unlock()

	Error(closeLog())

	writers := []io.Writer{rw}

	if IsLogFileEnabled() {
		var err error

		fw, err = newFileWriter()
		if err != nil {
			return err
		}

		writers = append(writers, fw)
	}

	flags := 0
	if !IsLogJsonEnabled() {
		flags = log.Lmsgprefix

		if IsLogVerboseEnabled() {
			flags = flags | log.LstdFlags | log.Lmicroseconds
		}
	}

	LogDebug = log.New(MultiWriter(append([]io.Writer{os.Stdout}, writers...)...), prefix(LevelDebug), flags)
	LogInfo = log.New(MultiWriter(append([]io.Writer{os.Stdout}, writers...)...), prefix(LevelInfo), flags)
	LogWarn = log.New(MultiWriter(append([]io.Writer{os.Stdout}, writers...)...), prefix(LevelWarn), flags)
	LogError = log.New(MultiWriter(append([]io.Writer{os.Stderr}, writers...)...), prefix(LevelError), flags)
	LogFatal = log.New(MultiWriter(append([]io.Writer{os.Stderr}, writers...)...), prefix(LevelFatal), flags)

	log.SetFlags(flags)

	ClearLogs()

	isLogInit = true

	return nil
}

func closeLog() error {
	if fw != nil {
		err := fw.closeFile()
		if err != nil {
			return err
		}
	}

	return nil
}

func InitTesting(t *testing.T) {
	tt = t

	Panic(initLog())
}

type testingT interface {
	Logf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

type entry struct {
	Timestamp  string `json:"timestamp"`
	Level      string `json:"level"`
	Source     string `json:"source"`
	Message    string `json:"message"`
	Stacktrace string `json:"stacktrace"`
}

func formatLog(level string, index int, msg string, addStacktrace bool) string {
	ri := GetRuntimeInfo(index)

	source := fmt.Sprintf("%s/%s:%d/%s", ri.Pack, ri.File, ri.Line, ri.Fn)

	switch {
	case IsLogJsonEnabled():
		e := entry{
			Timestamp:  time.Now().Format(time.RFC3339),
			Level:      level,
			Source:     source,
			Message:    msg,
			Stacktrace: "",
		}

		if addStacktrace {
			e.Stacktrace = ri.Stack
		}

		ba, _ := json.MarshalIndent(e, "", "  ")

		return string(ba)

	case IsLogVerboseEnabled():
		maxLen := 40
		if len(source) > maxLen {
			source = source[len(source)-maxLen:]
		}

		msg = fmt.Sprintf("%-"+strconv.Itoa(maxLen)+"s %s", source, msg)
	}

	if addStacktrace {
		msg = msg + "\n" + ri.Stack
	}

	return msg
}

func logDebugPrint(s string) {
	if testT.IsSet() {
		testT.Get().Logf(s)

		return
	}

	if time.Since(lastLog) > MillisecondToDuration(*FlagLogGap) {
		msg := fmt.Sprintf("time gap [%v]", time.Since(lastLog).Truncate(time.Millisecond))
		msg = fmt.Sprintf("%s %s -", strings.Repeat("-", 120-len(msg)-6-3), msg)

		LogDebug.Print(msg)
	}

	lastLog = time.Now()

	LogDebug.Print(s)
}

func logInfoPrint(s string) {
	if testT.IsSet() {
		testT.Get().Logf(s)

		return
	}

	LogInfo.Print(s)
}

func logWarnPrint(s string) {
	if testT.IsSet() {
		testT.Get().Logf(s)

		return
	}

	LogWarn.Print(s)
}

func logErrorPrint(s string) {
	if testT.IsSet() {
		testT.Get().Fatalf(s)

		return
	}

	LogError.Print(s)
}

func logFatalPrint(s string) {
	if testT.IsSet() {
		testT.Get().Fatalf(s)

		return
	}

	LogFatal.Print(s)
}

func Debug(format string, args ...any) {
	if !IsLogVerboseEnabled() {
		return
	}

	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	mu.Lock()
	defer mu.Unlock()

	logDebugPrint(formatLog(LevelDebug, 2, strings.TrimSpace(format), false))
}

func DebugIndex(index int, format string, args ...any) {
	if !IsLogVerboseEnabled() {
		return
	}

	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	mu.Lock()
	defer mu.Unlock()

	logDebugPrint(formatLog(LevelDebug, 2+index, strings.TrimSpace(format), false))
}

func DebugFunc(args ...any) {
	if !IsLogVerboseEnabled() {
		return
	}

	ri := GetRuntimeInfo(1)

	var str string

	switch len(args) {
	case 0:
		str = strings.TrimSpace(ri.Fn + "()")
	case 1:
		str = strings.TrimSpace(fmt.Sprintf(ri.Fn+"(): %v", args[0]))
	default:
		str = strings.TrimSpace(fmt.Sprintf(ri.Fn+"(): "+fmt.Sprintf("%v", args[0]), args[1:]...))
	}

	logDebugPrint(formatLog(LevelDebug, 2, str, false))
}

func Info(format string, args ...any) {
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	logInfoPrint(formatLog(LevelInfo, 2, strings.TrimSpace(format), false))
}

func Warn(format string, args ...any) {
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	logWarnPrint(formatLog(LevelWarn, 2, strings.TrimSpace(format), false))
}

func TraceError(err error) error {
	Error(err)

	return err
}

func IgnoreError(err error) bool {
	return err != nil
}

func DebugError(err error) bool {
	if !IsLogVerboseEnabled() || err == nil || IsErrExit(err) {
		return err != nil
	}

	mu.Lock()
	defer mu.Unlock()

	if err.Error() != lastErr {
		logDebugPrint(formatLog(LevelDebug, 2, strings.TrimSpace(err.Error()), IsLogVerboseEnabled()))
	}

	return true
}

func DebugErrorIndex(index int, err error) bool {
	if !IsLogVerboseEnabled() || err == nil || IsErrExit(err) {
		return err != nil
	}

	mu.Lock()
	defer mu.Unlock()

	if err.Error() != lastErr {
		logDebugPrint(formatLog(LevelDebug, 2+index, strings.TrimSpace(err.Error()), IsLogVerboseEnabled()))
	}

	return true
}

func WarnError(err error) bool {
	if err == nil || IsErrExit(err) {
		return err != nil
	}

	mu.Lock()
	defer mu.Unlock()

	if err.Error() != lastErr {
		logWarnPrint(formatLog(LevelWarn, 2, strings.TrimSpace(err.Error()), IsLogVerboseEnabled()))
	}

	return true
}

func Error(err error) bool {
	if err == nil || IsErrExit(err) {
		return err != nil
	}

	if IsSuppressedError(err) {
		return DebugErrorIndex(1, err)
	}

	mu.Lock()
	defer mu.Unlock()

	if err.Error() != lastErr {
		logErrorPrint(formatLog(LevelError, 2, strings.TrimSpace(err.Error()), IsLogVerboseEnabled()))
	}

	lastErr = err.Error()

	if *FlagLogBreak {
		Exit(1)
	}

	return true
}

func Panic(err error) {
	if err == nil || IsErrExit(err) {
		return
	}

	if err.Error() != lastErr {
		if isLogInit {
			if err.Error() != lastErr {
				logFatalPrint(formatLog(LevelFatal, 2, strings.TrimSpace(err.Error()), IsLogVerboseEnabled()))
			}
		} else {
			logFatalPrint(Capitalize(strings.TrimSpace(err.Error())))
		}
	}

	Exit(1)
}

func ClearLogs() {
	rw.Clearlogs()
}

func GetLogs() []string {
	return rw.GetLogs()
}
