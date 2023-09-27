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
	"sync"
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
)

const (
	PrefixDebug = "DEBUG "
	PrefixInfo  = "INFO  "
	PrefixWarn  = "WARN  "
	PrefixError = "ERROR "
	PrefixFatal = "FATAL "
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

	mu         sync.Mutex
	fw         *fileWriter
	rw                     = newMemoryWriter()
	LogDebug   *log.Logger = log.New(rw, prefix(PrefixDebug), 0)
	LogInfo    *log.Logger = log.New(rw, prefix(PrefixInfo), 0)
	LogWarn    *log.Logger = log.New(rw, prefix(PrefixWarn), 0)
	LogError   *log.Logger = log.New(rw, prefix(PrefixError), 0)
	LogFatal   *log.Logger = log.New(rw, prefix(PrefixFatal), 0)
	lastErr    string
	onceInit   sync.Once
	isDisabled bool
	testT      testingT
)

func isVerboseEnabled() bool {
	if *FlagLogVerbose {
		return true
	}

	for _, arg := range os.Args {
		if arg == "-"+FlagNameLogVerbose || arg == "-"+FlagNameLogVerbose+"=true" {
			return true
		}
	}

	return false
}

func isJsonEnabled() bool {
	if *FlagLogJson {
		return true
	}

	for _, arg := range os.Args {
		if arg == "-"+FlagNameLogJson || arg == "-"+FlagNameLogJson+"=true" {
			return true
		}
	}

	return false
}

func prefix(p string) string {
	if isVerboseEnabled() && !isJsonEnabled() {
		return p
	}

	return ""
}

func initLog() error { //FIXME
	mu.Lock()
	defer mu.Unlock()

	closeLog()

	writers := []io.Writer{rw}

	if FlagLogFileName != nil && *FlagLogFileName != "" {
		var err error

		fw, err = newFileWriter()
		if err != nil {
			return err
		}

		writers = append(writers, fw)
	}

	f := 0
	if !*FlagLogJson {
		f = log.Lmsgprefix
		if isVerboseEnabled() {
			f = f | log.LstdFlags | log.LUTC | log.Lmicroseconds
		}
	}

	LogDebug = log.New(io.MultiWriter(append([]io.Writer{os.Stdout}, writers...)...), prefix(PrefixDebug), f)
	LogInfo = log.New(io.MultiWriter(append([]io.Writer{os.Stdout}, writers...)...), prefix(PrefixInfo), f)
	LogWarn = log.New(io.MultiWriter(append([]io.Writer{os.Stdout}, writers...)...), prefix(PrefixWarn), f)
	LogError = log.New(io.MultiWriter(append([]io.Writer{os.Stderr}, writers...)...), prefix(PrefixError), f)
	LogFatal = log.New(io.MultiWriter(append([]io.Writer{os.Stderr}, writers...)...), prefix(PrefixFatal), f)

	log.SetFlags(f)

	onceInit.Do(func() {
		Events.AddListener(EventShutdown{}, func(event Event) {
			Error(closeLog())
		})

		Events.AddListener(EventFlagsParsed{}, func(event Event) {
			if *FlagLogSys && IsLinux() && !IsRunningInteractive() {
				// with SYSTEMD everything which is printed to console is automatically printed to journalctl

				*FlagLogSys = false
			}
		})

		msgs := append([]string{}, rw.msgs...)

		rw.Clear()

		for _, m := range msgs {
			switch {
			case strings.HasPrefix(m, PrefixDebug):
				logDebugPrint(m[len(PrefixDebug):])
			case strings.HasPrefix(m, PrefixInfo):
				logInfoPrint(m[len(PrefixInfo):])
			case strings.HasPrefix(m, PrefixWarn):
				logWarnPrint(m[len(PrefixWarn):])
			case strings.HasPrefix(m, PrefixError):
				logErrorPrint(m[len(PrefixError):])
			case strings.HasPrefix(m, PrefixFatal):
				logFatalPrint(m[len(PrefixFatal):])
			}
		}
	})

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
	testT = t
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
	//msg = Capitalize(msg)

	ri := GetRuntimeInfo(index)

	source := fmt.Sprintf("%s/%s:%d/%s", ri.Pack, ri.File, ri.Line, ri.Fn)

	if *FlagLogJson {
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

		ba, _ := json.Marshal(e)

		return string(ba)
	}

	if isVerboseEnabled() {
		max := 40
		if len(source) > max {
			source = source[len(source)-max:]
		}

		msg = fmt.Sprintf("%-"+strconv.Itoa(max)+"s %s", source, msg)
	}

	if addStacktrace {
		msg = msg + "\n" + ri.Stack
	}

	return msg
}

func logDebugPrint(s string) {
	if testT != nil {
		testT.Logf(s)

		return
	}

	LogDebug.Print(s)
}

func logInfoPrint(s string) {
	if testT != nil {
		testT.Logf(s)

		return
	}

	LogInfo.Print(s)
}

func logWarnPrint(s string) {
	if testT != nil {
		testT.Logf(s)

		return
	}

	LogWarn.Print(s)
}

func logErrorPrint(s string) {
	if testT != nil {
		testT.Fatalf(s)

		return
	}

	LogError.Print(s)
}

func logFatalPrint(s string) {
	if testT != nil {
		testT.Fatalf(s)

		return
	}

	logDebugPrint(s)
}

func Debug(format string, args ...any) {
	if isDisabled || !isVerboseEnabled() {
		return
	}

	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	logDebugPrint(formatLog(PrefixDebug, 2, strings.TrimSpace(format), false))
}

func DebugFunc(args ...any) {
	if isDisabled || !isVerboseEnabled() {
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

	logDebugPrint(formatLog(PrefixDebug, 2, str, false))
}

func Info(format string, args ...any) {
	if isDisabled {
		return
	}

	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	logInfoPrint(formatLog(PrefixInfo, 2, strings.TrimSpace(format), false))
}

func Warn(format string, args ...any) {
	if isDisabled {
		return
	}

	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	logWarnPrint(formatLog(PrefixWarn, 2, strings.TrimSpace(format), false))
}

func TraceError(err error) error {
	if isDisabled {
		return err
	}

	Error(err)

	return err
}

func DebugError(err error) bool {
	if isDisabled || err == nil || IsErrExit(err) {
		return err != nil
	}

	mu.Lock()
	defer mu.Unlock()

	if err.Error() != lastErr {
		logDebugPrint(formatLog(PrefixDebug, 2, strings.TrimSpace(err.Error()), isVerboseEnabled()))
	}

	return true
}

func WarnError(err error) bool {
	if isDisabled || err == nil || IsErrExit(err) {
		return err != nil
	}

	mu.Lock()
	defer mu.Unlock()

	if err.Error() != lastErr {
		logWarnPrint(formatLog(PrefixWarn, 2, strings.TrimSpace(err.Error()), isVerboseEnabled()))
	}

	return true
}

func Error(err error) bool {
	if isDisabled || err == nil || IsErrExit(err) {
		return err != nil
	}

	mu.Lock()
	defer mu.Unlock()

	if err.Error() != lastErr {
		logErrorPrint(formatLog(PrefixError, 2, strings.TrimSpace(err.Error()), isVerboseEnabled()))
	}

	lastErr = err.Error()

	return true
}

func Panic(err error) {
	if err == nil || IsErrExit(err) {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if err.Error() != lastErr {
		logFatalPrint(formatLog(PrefixFatal, 2, strings.TrimSpace(err.Error()), isVerboseEnabled()))
	}

	Exit(1)
}

func ClearLogs() {
	rw.Clear()
}

func GetLogs(w io.Writer) error {
	return rw.Copy(w)
}
