package common

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode"
)

var (
	onceShutdownHooks sync.Once
	shutdownHooks     []func()
)

const (
	MaxUint = ^uint(0)
	MinUint = 0
	MaxInt  = int(MaxUint >> 1)
	MinInt  = -MaxInt - 1
)

type MultiValueFlag []string

func (this *MultiValueFlag) String() string {
	if this == nil {
		return ""
	}

	return strings.Join(*this, ",")
}

func (this *MultiValueFlag) Set(value string) error {
	splits := strings.Split(value, ",")
	*this = append(*this, splits...)

	return nil
}

func init() {
	shutdownHooks = make([]func(), 0)
}

type ChannelError struct {
	m sync.Mutex
	l []error
}

func (c *ChannelError) Add(err error) {
	c.m.Lock()
	c.l = append(c.l, err)
	c.m.Unlock()
}

func (c *ChannelError) Get() error {
	c.m.Lock()
	defer c.m.Unlock()

	if len(c.l) > 0 {
		return c.l[0]
	} else {
		return nil
	}
}

func (c *ChannelError) GetAll() []error {
	return c.l
}

func (c *ChannelError) Exists() bool {
	c.m.Lock()
	defer c.m.Unlock()

	return len(c.l) > 0
}

// Exit exist app and run all registered shutdown hooks
func Done() {
	onceDone.Do(func() {
		onceShutdownHooks.Do(func() {
			for _, f := range shutdownHooks {
				f()
			}
		})

		closeLogFile()
	})
}

func AddShutdownHook(f func()) {
	shutdownHooks = append(shutdownHooks, nil)
	copy(shutdownHooks[1:], shutdownHooks[0:])
	shutdownHooks[0] = f
}

func AppFilename(newExt string) string {
	filename := Title()
	ext := filepath.Ext(filename)

	if len(ext) > 0 {
		filename = string(filename[:len(filename)-len(ext)])
	}

	return filename + newExt
}

func Title() string {
	path, err := os.Executable()
	if err != nil {
		path = os.Args[0]
	}

	path = filepath.Base(path)
	path = path[0:(len(path) - len(filepath.Ext(path)))]

	runes := []rune(path)
	for len(runes) > 0 && !unicode.IsLetter(runes[0]) {
		runes = runes[1:]
	}

	title := string(runes)

	DebugFunc(title)

	return title
}

func Version(major bool, minor bool, patch bool) string {
	if strings.Count(app.Version, ".") == 2 {
		s := strings.Split(app.Version, ".")

		sb := strings.Builder{}

		if major {
			sb.WriteString(s[0])
		}

		if minor {
			if sb.Len() > 0 {
				sb.WriteString(".")
			}

			sb.WriteString(s[1])
		}

		if patch {
			if sb.Len() > 0 {
				sb.WriteString(".")
			}

			sb.WriteString(s[2])
		}

		return sb.String()
	}

	return ""
}

func TitleVersion(major bool, minor bool, patch bool) string {
	return Title() + "-" + Version(major, minor, patch)
}

// IsWindowsOS reports true if underlying OS is MS Windows
func IsWindowsOS() bool {
	return runtime.GOOS == "windows"
}

// IsLinuxOS reports true if underlying OS is Linux
func IsLinuxOS() bool {
	return runtime.GOOS == "linux"
}

// IsMacOS reports true if underlying OS is MacOS
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// ToBool reports if value indicates "true"
func ToBool(s string) bool {
	if len(s) == 0 {
		return false
	}

	s = strings.ToLower(s)

	return s == "true" || s == "1" || strings.HasPrefix(s, "t") || strings.HasPrefix(s, "y") || strings.HasPrefix(s, "j")
}

func Eval(b bool, trueFunc interface{}, falseFunc interface{}) interface{} {
	if b {
		if f, ok := trueFunc.(func() interface{}); ok {
			return f()
		} else {
			return trueFunc
		}
	} else {
		if f, ok := falseFunc.(func() interface{}); ok {
			return f()
		} else {
			return falseFunc
		}
	}
}

func Min(v ...int) int {
	var r int
	for i, vi := range v {
		if i == 0 {
			r = vi
		} else {
			if vi < r {
				r = vi
			}
		}
	}

	return r
}

func Max(v ...int) int {
	var r int
	for i, vi := range v {
		if i == 0 {
			r = vi
		} else {
			if vi > r {
				r = vi
			}
		}
	}

	return r
}
