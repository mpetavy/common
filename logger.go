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
				LogDebug.Print(m[len(PrefixDebug):])
			case strings.HasPrefix(m, PrefixInfo):
				LogInfo.Print(m[len(PrefixInfo):])
			case strings.HasPrefix(m, PrefixWarn):
				LogInfo.Print(m[len(PrefixWarn):])
			case strings.HasPrefix(m, PrefixError):
				LogInfo.Print(m[len(PrefixError):])
			case strings.HasPrefix(m, PrefixFatal):
				LogInfo.Print(m[len(PrefixFatal):])
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

type entry struct {
	Timestamp  string `json:"timestamp"`
	Level      string `json:"level"`
	Message    string `json:"message"`
	Stacktrace string `json:"stacktrace"`
}

func formatLog(level string, index int, msg string, addStacktrace bool) string {
	//msg = Capitalize(msg)

	ri := GetRuntimeInfo(index)

	if *FlagLogJson {
		e := entry{
			Timestamp:  time.Now().Format(time.RFC3339),
			Level:      level,
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
		prefix := fmt.Sprintf("%s/%s:%d/%s", ri.Pack, ri.File, ri.Line, ri.Fn)

		max := 40
		if len(prefix) > max {
			prefix = prefix[len(prefix)-max:]
		}

		msg = fmt.Sprintf("%-"+strconv.Itoa(max)+"s %s", prefix, msg)
	}

	if addStacktrace {
		msg = msg + "\n" + ri.Stack
	}

	return msg
}

func Debug(format string, args ...any) {
	if isDisabled || !isVerboseEnabled() {
		return
	}

	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	LogDebug.Printf(formatLog(PrefixDebug, 2, strings.TrimSpace(format), false))
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

	LogDebug.Printf(formatLog(PrefixDebug, 2, str, false))
}

func Info(format string, args ...any) {
	if isDisabled {
		return
	}

	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	LogInfo.Printf(formatLog(PrefixInfo, 2, strings.TrimSpace(format), false))
}

func Warn(format string, args ...any) {
	if isDisabled {
		return
	}

	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}

	LogWarn.Printf(formatLog(PrefixWarn, 2, strings.TrimSpace(format), false))
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
		LogDebug.Printf(formatLog(PrefixDebug, 2, strings.TrimSpace(err.Error()), true))
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
		LogWarn.Printf(formatLog(PrefixWarn, 2, strings.TrimSpace(err.Error()), true))
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
		LogError.Printf(formatLog(PrefixError, 2, strings.TrimSpace(err.Error()), true))
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
		LogFatal.Printf(formatLog(PrefixFatal, 2, strings.TrimSpace(err.Error()), true))
	}

	Exit(1)
}

func ClearLogs() error {
	rw.Clear()

	return nil
}

func GetLogs(w io.Writer) error {
	return rw.Copy(w)
}
