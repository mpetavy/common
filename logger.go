package common

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	FlagNameLogFileName     = "log.file"
	FlagNameLogFileSize     = "log.filesize"
	FlagNameLogVerbose      = "log.verbose"
	FlagNameLogVerboseError = "log.verbose.error"
	FlagNameLogIO           = "log.io"
	FlagNameLogJson         = "log.json"
	FlagNameLogSys          = "log.sys"
	FlagNameLogCount        = "log.count"
	FlagNameLogBreakOnError = "log.breakonerror"
	FlagNameLogGap          = "log.gap"
	FlagNameLogEqualError   = "log.equalerror"
)

const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
	LevelFatal = "FATAL"
)

var (
	FlagLogFileName     = SystemFlagString(FlagNameLogFileName, "", "filename to log file")
	FlagLogFileSize     = SystemFlagInt(FlagNameLogFileSize, 5*1024*1024, "max log file size")
	FlagLogVerbose      = flag.Bool(FlagNameLogVerbose, false, "verbose logging")
	FlagLogVerboseError = SystemFlagBool(FlagNameLogVerboseError, false, "verbose error logging")
	FlagLogIO           = SystemFlagBool(FlagNameLogIO, false, "trace logging")
	FlagLogJson         = SystemFlagBool(FlagNameLogJson, false, "JSON output")
	FlagLogSys          = SystemFlagBool(FlagNameLogSys, false, "Use OS system logger")
	FlagLogCount        = SystemFlagInt(FlagNameLogCount, 1000, "log count")
	FlagLogBreakOnError = SystemFlagString(FlagNameLogBreakOnError, "", "break on logging an error")
	FlagLogGap          = SystemFlagInt(FlagNameLogGap, 100, "time gap after show a separator")
	FlagLogEqualError   = SystemFlagBool(FlagNameLogEqualError, false, "Log equal (repeated) error")

	// synchronizes logging output
	logMutex    ReentrantMutex
	fw          *fileWriter
	rw                      = newMemoryWriter()
	LogDebug    *log.Logger = log.New(rw, prefix(LevelDebug), 0)
	LogInfo     *log.Logger = log.New(rw, prefix(LevelInfo), 0)
	LogWarn     *log.Logger = log.New(rw, prefix(LevelWarn), 0)
	LogError    *log.Logger = log.New(os.Stderr, prefix(LevelError), 0)
	LogFatal    *log.Logger = log.New(os.Stderr, prefix(LevelFatal), 0)
	lastLogTime             = time.Now()
	isLogInit   bool
)

type EventLog struct {
	Entry *LogEntry
}

func NewLogEntry(level string, msg string, ri RuntimeInfo) *LogEntry {
	now := time.Now().UTC()

	return &LogEntry{
		Time:          now,
		Timestamp:     now.Format(time.RFC3339),
		GoRoutineId:   GoRoutineId(),
		Level:         level,
		Source:        fmt.Sprintf("%s/%s/%s:%d", ri.Pack, ri.File, ri.Fn, ri.Line),
		RuntimeInfo:   ri,
		StacktraceMsg: msg + "\n" + ri.Stack,
		Msg:           msg,
	}
}

type LogEntry struct {
	Time          time.Time   `json:"-"`
	Timestamp     string      `json:"timestamp"`
	GoRoutineId   uint64      `json:"goRoutineId"`
	Level         string      `json:"level"`
	Source        string      `json:"source"`
	RuntimeInfo   RuntimeInfo `json:"runtimeInfo"`
	Msg           string      `json:"msg"`
	StacktraceMsg string      `json:"-"`
	PrintMsg      string      `json:"-"`
}

func init() {
	Events.AddListener(EventFlags{}, func(event Event) {
		if *FlagLogSys && IsLinux() && !IsRunningInteractive() {
			// with SYSTEMD everything which is printed to console is automatically printed to journalctl

			*FlagLogSys = false
		}
	})
}

func IsLogVerboseEnabled() bool {
	// use isLogInit (initial = false) to capture all DEBUG logs before first initLogging so can we log them if verbose logging is enabled

	if FlagLogVerbose != nil {
		return *FlagLogVerbose || !isLogInit
	}

	return IsFlagProvided(FlagNameLogVerbose) || !isLogInit
}

func IsLogJsonEnabled() bool {
	if FlagLogJson != nil {
		return *FlagLogJson
	}

	return IsFlagProvided(FlagNameLogJson)
}

func IsLogFileEnabled() bool {
	if FlagLogFileName != nil {
		return *FlagLogFileName != ""
	}

	return IsFlagProvided(FlagNameLogFileName)
}

func prefix(p string) string {
	if IsLogVerboseEnabled() && !IsLogJsonEnabled() {
		return fmt.Sprintf("%-6s", p)
	}

	return ""
}

func InitLog() error {
	if !logMutex.TryLock() {
		return fmt.Errorf("cannot reentrant lock")
	}
	defer logMutex.Unlock()

	Error(closeLog())

	writers := []io.Writer{rw}

	if IsLogFileEnabled() {
		err := CheckOutputPath(filepath.Dir(*FlagLogFileName))
		if err != nil {
			return err
		}

		fw, err = newFileWriter()
		if err != nil {
			return err
		}

		writers = append(writers, fw)
	}

	flags := 0
	if !IsLogJsonEnabled() {
		flags = log.Lmsgprefix
	}

	LogDebug = log.New(MultiWriter(append([]io.Writer{os.Stdout}, writers...)...), "", flags)
	LogInfo = log.New(MultiWriter(append([]io.Writer{os.Stdout}, writers...)...), "", flags)
	LogWarn = log.New(MultiWriter(append([]io.Writer{os.Stdout}, writers...)...), "", flags)
	LogError = log.New(MultiWriter(append([]io.Writer{os.Stderr}, writers...)...), "", flags)
	LogFatal = log.New(MultiWriter(append([]io.Writer{os.Stderr}, writers...)...), "", flags)

	log.SetFlags(flags)

	isLogInit = true

	if *FlagLogVerbose {
		for _, line := range GetLogs() {
			LogDebug.Print(line[len(LevelDebug)+1:])
		}
	}

	ClearLogs()

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

func formatLog(level string, index int, msg string, addStacktrace bool) *LogEntry {
	verbose := IsLogVerboseEnabled() || (*FlagLogVerboseError && slices.Contains([]string{LevelError, LevelFatal}, level))

	logEntry := NewLogEntry(level, msg, GetRuntimeInfo(index))

	if addStacktrace || ((level == LevelError || level == LevelFatal) && App() != nil && App().StartFunc != nil && App().StopFunc != nil) {
		msg = logEntry.StacktraceMsg
	}

	// shorten the "source" position only for console log

	source := logEntry.Source
	maxLen := 40
	if len(source) > maxLen {
		source = source[len(source)-maxLen:]
	}

	switch {
	case IsLogJsonEnabled():
		ba, _ := json.MarshalIndent(logEntry, "", "  ")

		msg = string(ba)
	case verbose:
		if level == LevelDebug {
			msg = strings.ReplaceAll(msg, "\n\t", "\n")
			msg = strings.ReplaceAll(msg, "\n", "\n\t")
		}

		msg = fmt.Sprintf("%s | %9d | %-5s | %-"+strconv.Itoa(maxLen)+"s | %s", logEntry.Time.Format(SortedDateTimeMilliMask), logEntry.GoRoutineId, level, source, msg)
	default:
		if level != LevelDebug && level != LevelInfo {
			msg = fmt.Sprintf("%s: %s", Capitalize(strings.ToLower(level)), msg)
		}
	}

	logEntry.PrintMsg = msg

	return logEntry
}

func logDebugPrint(s string) {
	if time.Since(lastLogTime) > MillisecondToDuration(*FlagLogGap) {
		msg := fmt.Sprintf("time gap [%v]", time.Since(lastLogTime).Truncate(time.Millisecond))
		msg = fmt.Sprintf("%s %s -", strings.Repeat("-", 120-len(msg)-6-3), msg)

		LogDebug.Print(msg)
	}

	lastLogTime = time.Now()

	LogDebug.Print(s)
}

func logInfoPrint(s string) {
	LogInfo.Print(s)
}

func logWarnPrint(s string) {
	LogWarn.Print(s)
}

func logErrorPrint(s string) {
	LogError.Print(s)
}

func logFatalPrint(s string) {
	LogFatal.Print(s)
}

func Debug(format string, args ...any) {
	if !IsLogVerboseEnabled() {
		return
	}

	if !logMutex.TryLock() {
		return
	}
	defer logMutex.Unlock()

	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	logEntry := formatLog(LevelDebug, 2, strings.TrimSpace(format), false)

	logDebugPrint(logEntry.PrintMsg)
}

func DebugIndex(index int, format string, args ...any) {
	if !IsLogVerboseEnabled() {
		return
	}

	if !logMutex.TryLock() {
		return
	}
	defer logMutex.Unlock()

	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	logEntry := formatLog(LevelDebug, 2+index, strings.TrimSpace(format), false)

	logDebugPrint(logEntry.PrintMsg)
}

func DebugFunc(args ...any) {
	if !IsLogVerboseEnabled() {
		return
	}

	if !logMutex.TryLock() {
		return
	}
	defer logMutex.Unlock()

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

	logEntry := formatLog(LevelDebug, 2, str, false)

	logDebugPrint(logEntry.PrintMsg)
}

func Info(format string, args ...any) {
	if !logMutex.TryLock() {
		return
	}
	defer logMutex.Unlock()

	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	logEntry := formatLog(LevelInfo, 2, strings.TrimSpace(format), false)

	Events.Emit(EventLog{Entry: logEntry}, false)

	logInfoPrint(logEntry.PrintMsg)
}

func Warn(format string, args ...any) {
	if !logMutex.TryLock() {
		return
	}
	defer logMutex.Unlock()

	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	logEntry := formatLog(LevelWarn, 2, strings.TrimSpace(format), false)

	Events.Emit(EventLog{Entry: logEntry}, false)

	logWarnPrint(logEntry.PrintMsg)
}

func TraceError(err error) error {
	Error(err)

	return err
}

func IgnoreError(err error) bool {
	return err != nil
}

func DebugError(err error) bool {
	if err == nil || !IsLogVerboseEnabled() || IsErrExit(err) {
		return err != nil
	}

	if !logMutex.TryLock() {
		return err != nil
	}
	defer logMutex.Unlock()

	logEntry := formatLog(LevelDebug, 2, strings.TrimSpace(err.Error()), IsLogVerboseEnabled())

	logDebugPrint(logEntry.PrintMsg)

	return true
}

func DebugErrorIndex(index int, err error) bool {
	if err == nil || !IsLogVerboseEnabled() || IsErrExit(err) {
		return err != nil
	}

	if !logMutex.TryLock() {
		return err != nil
	}
	defer logMutex.Unlock()

	logEntry := formatLog(LevelDebug, 2+index, strings.TrimSpace(err.Error()), IsLogVerboseEnabled())

	logDebugPrint(logEntry.PrintMsg)

	return true
}

func WarnError(err error) bool {
	if err == nil || IsErrExit(err) {
		return err != nil
	}

	if !logMutex.TryLock() {
		return err != nil
	}
	defer logMutex.Unlock()

	if IsSuppressedError(err) {
		return DebugErrorIndex(1, err)
	}

	logEntry := formatLog(LevelWarn, 2, strings.TrimSpace(err.Error()), IsLogVerboseEnabled())

	Events.Emit(EventLog{Entry: logEntry}, false)

	logWarnPrint(logEntry.PrintMsg)

	return true
}

func isLikeLastError(logEntry *LogEntry) bool {
	if *FlagLogEqualError {
		return false
	}

	entry, ok := GoRoutineVars.Get().Get(goVarslastLogEntry)
	if !ok {
		return false
	}

	lastErrorEntry := entry.(*LogEntry)

	b := lastErrorEntry.Msg == logEntry.Msg && logEntry.Time.Sub(lastErrorEntry.Time) <= (time.Millisecond*50)

	return b
}

func LastError() *LogEntry {
	entry, ok := GoRoutineVars.Get().Get(goVarslastLogEntry)
	if !ok {
		return nil
	}

	return entry.(*LogEntry)
}

func Error(err error) bool {
	if err == nil || IsErrExit(err) {
		return err != nil
	}

	if !logMutex.TryLock() {
		return err != nil
	}
	defer logMutex.Unlock()

	if IsSuppressedError(err) {
		return DebugErrorIndex(1, err)
	}

	logEntry := formatLog(LevelError, 2, strings.TrimSpace(err.Error()), IsLogVerboseEnabled())

	if isLikeLastError(logEntry) {
		return true
	}

	Events.Emit(EventLog{Entry: logEntry}, false)

	GoRoutineVars.Get().Set(goVarslastLogEntry, logEntry)

	logErrorPrint(logEntry.PrintMsg)

	if *FlagLogBreakOnError != "" && (ToBool(*FlagLogBreakOnError) || strings.Contains(logEntry.PrintMsg, *FlagLogBreakOnError)) {
		Exit(1)
	}

	return true
}

func Panic(err error) {
	if err == nil || IsErrExit(err) {
		return
	}

	if !logMutex.TryLock() {
		return
	}
	defer logMutex.Unlock()

	logEntry := formatLog(LevelFatal, 2, strings.TrimSpace(err.Error()), IsLogVerboseEnabled())

	Events.Emit(EventLog{Entry: logEntry}, false)

	if isLogInit {
		logFatalPrint(logEntry.PrintMsg)
	} else {
		logFatalPrint(Capitalize(strings.TrimSpace(err.Error())))
	}

	Exit(1)
}

func ClearLogs() {
	rw.Clearlogs()
}

func GetLogs() []string {
	return rw.GetLogs()
}

func LevelToIndex(level string) int {
	return slices.Index([]string{
		LevelDebug,
		LevelInfo,
		LevelWarn,
		LevelError,
		LevelFatal,
	}, level)
}

func StartInfo(msg string) {
	Info("Start %s", msg)
}

func StopInfo(msg string) {
	Info("Stop %s", msg)
}
