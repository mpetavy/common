package common

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// https://medium.com/@vCabbage/go-timeout-commands-with-os-exec-commandcontext-ba0c861ed738

type ErrWatchdog struct {
	msg string
}

func (e *ErrWatchdog) Error() string {
	return e.msg
}

func WatchdogCmd(cmd *exec.Cmd, timeout time.Duration) error {
	doneCh := make(chan error)

	start := time.Now()

	err := cmd.Start()
	if err != nil {
		return err
	}

	Debug("Watchdog observe process pid: %d timeout: %v cmd: %s ...", cmd.Process.Pid, timeout, CmdToString(cmd))

	go func() {
		doneCh <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		Debug("Watchdog: process killed! pid: %d timeout: %v cmd: %s time: %v", cmd.Process.Pid, timeout, CmdToString(cmd), time.Since(start))
		Error(cmd.Process.Kill())

		return &ErrWatchdog{msg: fmt.Sprintf("Watchdog killed process pid: %d cmd: %s after: %v", cmd.Process.Pid, CmdToString(cmd), time.Since(start))}
	case err = <-doneCh:
		exitcode := 0
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				exitcode = exitError.ExitCode()
			} else {
				exitcode = -1
			}
		}

		exitstate := ""
		switch exitcode {
		case 0:
			exitstate = "successfull"
		default:
			exitstate = "failed"
		}

		output := "<na>"
		bu, ok := cmd.Stdout.(*bytes.Buffer)
		if ok {
			output = "\n" + string(bu.Bytes())
		}

		Debug("Watchdog: process %s! pid: %d exitcode: %d timeout: %v cmd: %s time: %s output: %s", exitstate, cmd.Process.Pid, exitcode, timeout, CmdToString(cmd), time.Since(start), output)

		return err
	}
}

func WatchdogFunc(msg string, fn func() error, timeout time.Duration) error {
	doneCh := make(chan error)

	start := time.Now()

	var err error

	go func() {
		doneCh <- fn()
	}()

	select {
	case <-time.After(timeout):
		Debug("Watchdog: function killed! time: %v", time.Since(start))

		return &ErrWatchdog{msg: msg}
	case err = <-doneCh:
		exitstate := ""
		if err != nil {
			exitstate = "failed"
		} else {
			exitstate = "successfull"
		}

		Debug("Watchdog: function %s! time: %s", exitstate, time.Since(start))
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
