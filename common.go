package common

import (
	"flag"
	"fmt"
	"runtime"
	"sync"

	"math/rand"
	"os/user"
	"strings"
	"time"
)

var (
	onceShutdownHooks sync.Once
	shutdownHooks     []func() error
	fnShutdownHooks   func()

	UserName    *string
	UserHomeDir *string
)

func init() {
	shutdownHooks = make([]func() error, 0)
	fnShutdownHooks = func() {
		for _, f := range shutdownHooks {
			err := f()
			if err != nil {
				Error(err)
			}
		}
	}

	UserName = new(string)
	UserHomeDir = new(string)

	usr, err := user.Current()
	if err == nil {
		*UserHomeDir = usr.HomeDir
	}

	UserName = flag.String("user.name", *UserName, "user name")
	UserHomeDir = flag.String("user.homedir", *UserHomeDir, "user home directory")
}

// Exit exist app and run all registered shutdown hooks
func Cleanup() {
	runShutdownHooks()
}

func runShutdownHooks() {
	onceShutdownHooks.Do(fnShutdownHooks)
}

func AddShutdownHook(f func() error) {
	shutdownHooks = append(shutdownHooks, nil)
	copy(shutdownHooks[1:], shutdownHooks[0:])
	shutdownHooks[0] = f
}

// IsWindowsOS reports true if underlying OS is MS Windows
func IsWindowsOS() bool {
	result := runtime.GOOS == "windows"

	return result
}

// IsLinuxOS reports true if underlying OS is Linux
func IsLinuxOS() bool {
	result := runtime.GOOS == "linux"

	return result
}

// IsMacOS reports true if underlying OS is MacOS
func IsMacOS() bool {
	result := runtime.GOOS == "darwin"

	return result
}

// IsAMD64 reports true if underlying OS is 64Bit ready
func IsAMD64() bool {
	result := runtime.GOARCH == "amd64"

	Debug(fmt.Sprintf("isAMD64 : %v", result))

	return result
}

func ToBool(s string) bool {
	if len(s) == 0 {
		return false
	}

	s = strings.ToLower(s)

	return s == "true" || s == "t"
}

// Translate a i18n message
func Translate(msg string, args ...interface{}) string {
	return fmt.Sprintf(msg, args...)
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
