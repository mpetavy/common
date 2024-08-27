package common

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

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

func Eval[T any](b bool, trueFunc T, falseFunc T) T {
	if b {
		return trueFunc
	} else {
		return falseFunc
	}
}

func Sleep(d time.Duration) {
	id := uuid.New().String()

	DebugIndex(1, "Sleep [%s] %v... ", id, d)

	select {
	case <-time.After(d):
		DebugIndex(1, "Sleep [%s] %v continue", id, d)
	case <-AppLifecycle().Channel():
		DebugIndex(1, "Sleep [%s] interrupted because of app lifecyle end", id)
	}
}

func SleepWithChannel(d time.Duration, ch chan struct{}) {
	id := uuid.New().String()

	select {
	case <-time.After(d):
		DebugIndex(1, "Sleep [%s] %v continue", id, d)
	case <-AppLifecycle().Channel():
		DebugIndex(1, "Sleep [%s] interrupted because of app lifecyle end", id)
	case <-ch:
		DebugIndex(1, "Sleep [%s] interrupted because of channel end", id)
	}
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

func RndBytes(n int) ([]byte, error) {
	DebugFunc()

	b := make([]byte, n)
	_, err := rand.Read(b)
	if Error(err) {
		return nil, err
	}

	return b, nil
}

func RndString(l int) (string, error) {
	DebugFunc()

	var letters = []rune("012345678abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	var lenLetter = len(letters)

	sb := strings.Builder{}
	for i := 0; i < l; i++ {
		sb.WriteRune(letters[Rnd(lenLetter)])
	}

	return sb.String(), nil
}

func ExecuteCmd(cmd *exec.Cmd) ([]byte, error) {
	Debug("exec: %s", SurroundWith(cmd.Args, "\""))

	ba, err := cmd.CombinedOutput()

	Debug("exec output --- start ---")
	Debug("\n%s", string(ba))
	Debug("exec output --- ende ---")
	Debug("exec exit code: %d", cmd.ProcessState.ExitCode())

	if Error(err) {
		return nil, fmt.Errorf("%s: %s", err.Error(), string(ba))
	}

	if !cmd.ProcessState.Success() {
		return nil, fmt.Errorf("exec exit error: %d", cmd.ProcessState.ExitCode())
	}

	return ba, nil
}

func RunScript(timeout time.Duration, filename string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer func() {
		cancel()
	}()

	var cmd *exec.Cmd

	if IsWindows() {
		cmd = exec.CommandContext(ctx, "cmd.exe", "/c", filename)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", "./"+filename)
	}

	ba, err := ExecuteCmd(cmd)
	if Error(err) {
		return nil, err
	}

	return ba, nil
}
