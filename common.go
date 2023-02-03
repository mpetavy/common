package common

import (
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/exp/constraints"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	MaxUint = ^uint(0)
	MaxInt  = int(MaxUint >> 1)
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

func Min[T constraints.Ordered](v ...T) T {
	var r T
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

func Max[T constraints.Ordered](v ...T) T {
	var r T
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

func Sleep(d time.Duration) {
	id := uuid.New().String()

	Debug("Sleep [%s] %v... ", id, d)

	time.Sleep(d)

	Debug("Sleep [%s] %v continue", id, d)
}

func CatchPanic(fn func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				err = fmt.Errorf(x)
			case error:
				err = x
			default:
				err = fmt.Errorf("unknown panic: %+v", r)
			}
		}
	}()

	fn()

	return nil
}

func ToSlice[T any](slice ...T) []T {
	return slice
}
