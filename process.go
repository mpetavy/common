package common

import (
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

func (e *ErrWatchdog) Error() string {
	return fmt.Sprintf("watchdog killed process pid: %d cmd: %s after: %v", e.Cmd.Process.Pid, ToString(*e.Cmd), time.Since(e.Start))
}

func Watchdog(cmd *exec.Cmd, timeout time.Duration) error {
	doneCh := make(chan error)

	start := time.Now()

	err := cmd.Start()
	if err != nil {
		return err
	}

	go func() {
		doneCh <- cmd.Wait()
	}()

	pid := cmd.Process.Pid

	if pid > 0 {
		ti := time.After(timeout)

		select {
		case <-ti:
			cmd.Process.Kill()

			return &ErrWatchdog{cmd.Process.Pid, start, cmd}
		case err = <-doneCh:
			Debug("watchdog finished process pid: %d cmd: %s\n time: %s", cmd.Process.Pid, ToString(*cmd), time.Since(start))
			return err
		}
	}

	return nil
}

func StillAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	} else {
		return process.Signal(syscall.Signal(0)) != nil
	}
}
