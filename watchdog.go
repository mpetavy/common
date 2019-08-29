package common

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// https://medium.com/@vCabbage/go-timeout-commands-with-os-exec-commandcontext-ba0c861ed738

type ErrWatchdog struct {
	Pid   int
	Start time.Time
	Cmd   *exec.Cmd
}

var (
	flagTimeout *time.Duration
)

func init() {
	flagTimeout = flag.Duration("watchdog.timeout", 0, "watchdog timeout")
}

func (e *ErrWatchdog) Error() string {
	return fmt.Sprintf("watchdog killed process pid: %d cmd: %s after: %v", e.Cmd.Process.Pid, ToString(*e.Cmd), time.Since(e.Start))
}

func Watchdog(cmd *exec.Cmd, timeout time.Duration) error {
	if *flagTimeout > timeout {
		timeout = *flagTimeout
	}

	doneCh := make(chan error)

	start := time.Now()

	err := cmd.Start()
	if err != nil {
		return err
	}

	go func() {
		doneCh <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		DebugError(cmd.Process.Kill())

		return &ErrWatchdog{cmd.Process.Pid, start, cmd}
	case err = <-doneCh:
		Debug("watchdog finished successfully observerd process pid: %d cmd: %s time: %s", cmd.Process.Pid, ToString(*cmd), time.Since(start))
		return err
	}
}

func StillAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	} else {
		return process.Signal(syscall.Signal(0)) != nil
	}
}
