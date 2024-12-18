package common

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
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
	FlagLogVerboseError = flag.Bool(FlagNameLogVerboseError, false, "verbose error logging")
	FlagLogIO           = SystemFlagBool(FlagNameLogIO, false, "trace logging")
	FlagLogJson         = SystemFlagBool(FlagNameLogJson, false, "JSON output")
	FlagLogSys          = SystemFlagBool(FlagNameLogSys, false, "Use OS system logger")
	FlagLogCount        = SystemFlagInt(FlagNameLogCount, 1000, "log count")
	FlagLogBreakOnError = SystemFlagString(FlagNameLogBreakOnError, "", "break on logging an error")
	FlagLogGap          = SystemFlagInt(FlagNameLogGap, 100, "time gap after show a separator")
	FlagLogEqualError   = SystemFlagBool(FlagNameLogEqualError, false, "Log equal (repeated) error")

	mu          ReentrantMutex
	fw          *fileWriter
	rw                      = newMemoryWriter()
	LogDebug    *log.Logger = log.New(rw, prefix(LevelDebug), 0)
	LogInfo     *log.Logger = log.New(rw, prefix(LevelInfo), 0)
	LogWarn     *log.Logger = log.New(rw, prefix(LevelWarn), 0)
	LogError    *log.Logger = log.New(os.Stderr, prefix(LevelError), 0)
	LogFatal    *log.Logger = log.New(os.Stderr, prefix(LevelFatal), 0)
	lastLogTime             = time.Now()
	isLogInit   bool
	listNoDebug = NewGoRoutinesRegister()
)

type EventLog struct {
	Entry *LogEntry
}

func NewLogEntry(level string, source string, msg string) *LogEntry {
	now := time.Now().UTC()

	return &LogEntry{
		Time:        now,
		Timestamp:   now.Format(time.RFC3339),
		GoRoutineId: GoRoutineId(),
		Level:       level,
		Source:      source,
		Msg:         msg,
	}
}

type LogEntry struct {
	Time        time.Time `json:"-"`
	Timestamp   string    `json:"timestamp"`
	GoRoutineId uint64    `json:"goRoutineId"`
	Level       string    `json:"level"`
	Source      string    `json:"source"`
	Msg         string    `json:"message"`
	PrintMsg    string    `json:"printMessage"`
}

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
	}

	LogDebug = log.New(MultiWriter(append([]io.Writer{os.Stdout}, writers...)...), "", flags)
	LogInfo = log.New(MultiWriter(append([]io.Writer{os.Stdout}, writers...)...), "", flags)
	LogWarn = log.New(MultiWriter(append([]io.Writer{os.Stdout}, writers...)...), "", flags)
	LogError = log.New(MultiWriter(append([]io.Writer{os.Stderr}, writers...)...), "", flags)
	LogFatal = log.New(MultiWriter(append([]io.Writer{os.Stderr}, writers...)...), "", flags)

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

func formatLog(level string, index int, msg string, addStacktrace bool) *LogEntry {
	verbose := IsLogVerboseEnabled() || (*FlagLogVerboseError && slices.Contains([]string{LevelError, LevelFatal}, level))

	ri := GetRuntimeInfo(index)

	source := fmt.Sprintf("%s/%s/%s:%d", ri.Pack, ri.File, ri.Fn, ri.Line)

	logEntry := NewLogEntry(level, source, msg)

	if addStacktrace || ((level == LevelError || level == LevelFatal) && App() != nil && App().StartFunc != nil && App().StopFunc != nil) {
		msg = msg + "\n" + ri.Stack
	}

	// shorten the "source" position only for console log

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
	if listNoDebug.IsRegistered() || !IsLogVerboseEnabled() {
		return
	}

	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	mu.Lock()
	defer mu.Unlock()

	logEntry := formatLog(LevelDebug, 2, strings.TrimSpace(format), false)

	logDebugPrint(logEntry.PrintMsg)
}

func DebugIndex(index int, format string, args ...any) {
	if listNoDebug.IsRegistered() || !IsLogVerboseEnabled() {
		return
	}

	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	mu.Lock()
	defer mu.Unlock()

	logEntry := formatLog(LevelDebug, 2+index, strings.TrimSpace(format), false)

	logDebugPrint(logEntry.PrintMsg)
}

func DebugFunc(args ...any) {
	if listNoDebug.IsRegistered() || !IsLogVerboseEnabled() {
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

	mu.Lock()
	defer mu.Unlock()

	logEntry := formatLog(LevelDebug, 2, str, false)

	logDebugPrint(logEntry.PrintMsg)
}

func Info(format string, args ...any) {
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	mu.Lock()
	defer mu.Unlock()

	logEntry := formatLog(LevelInfo, 2, strings.TrimSpace(format), false)

	Events.Emit(EventLog{Entry: logEntry}, false)

	logInfoPrint(logEntry.PrintMsg)
}

func Warn(format string, args ...any) {
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	mu.Lock()
	defer mu.Unlock()

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
	if err == nil || listNoDebug.IsRegistered() || !IsLogVerboseEnabled() || IsErrExit(err) {
		return err != nil
	}

	mu.Lock()
	defer mu.Unlock()

	logEntry := formatLog(LevelDebug, 2, strings.TrimSpace(err.Error()), IsLogVerboseEnabled())

	logDebugPrint(logEntry.PrintMsg)

	return true
}

func DebugErrorIndex(index int, err error) bool {
	if err == nil || listNoDebug.IsRegistered() || !IsLogVerboseEnabled() || IsErrExit(err) {
		return err != nil
	}

	mu.Lock()
	defer mu.Unlock()

	logEntry := formatLog(LevelDebug, 2+index, strings.TrimSpace(err.Error()), IsLogVerboseEnabled())

	logDebugPrint(logEntry.PrintMsg)

	return true
}

func WarnError(err error) bool {
	if err == nil || IsErrExit(err) {
		return err != nil
	}

	if IsSuppressedError(err) {
		return DebugErrorIndex(1, err)
	}

	mu.Lock()
	defer mu.Unlock()

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

func Error(err error) bool {
	if err == nil || IsErrExit(err) {
		return err != nil
	}

	if IsSuppressedError(err) {
		return DebugErrorIndex(1, err)
	}

	mu.Lock()
	defer mu.Unlock()

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
	fatal(err, 3)
}

func fatal(err error, index int) {
	if err == nil || IsErrExit(err) {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	logEntry := formatLog(LevelFatal, index, strings.TrimSpace(err.Error()), IsLogVerboseEnabled())

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

func NoDebug(fn func()) {
	listNoDebug.Register()
	defer func() {
		listNoDebug.Deregister()
	}()

	fn()
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
