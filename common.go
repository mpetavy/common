package common

import (
	"runtime"
	"sync"

	"math/rand"
	"strings"
	"time"
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

func Rnd(max int) int {
	rand.Seed(time.Now().UnixNano())

	return rand.Intn(max)
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
