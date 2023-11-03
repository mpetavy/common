package common

import (
	"crypto/rand"
	"embed"
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"os"
	"runtime"
	"strings"
	"time"
)

//go:embed embed/*
var embedfs embed.FS

// IsWindowsOS reports true if underlying OS is MS Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsLinuxOS reports true if underlying OS is Linux
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsMacOS reports true if underlying OS is MacOS
func IsMac() bool {
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

func Sleep(d time.Duration) {
	if !*FlagLogVerbose {
		time.Sleep(d)

		return
	}

	id := uuid.New().String()

	DebugIndex(1, "Sleep [%s] %v... ", id, d)

	time.Sleep(d)

	DebugIndex(1, "Sleep [%s] %v continue", id, d)
}

func Catch(fn func() error) (err error) {
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

	return fn()
}

func Exit(code int) {
	done()

	os.Exit(code)
}

func Rnd(max int) int {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	Panic(err)

	return int(nBig.Int64())
}
