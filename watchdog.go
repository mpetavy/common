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
	Msg string
}

func (e *ErrWatchdog) Error() string {
	return e.Msg
}

func NewWatchdogCmd(cmd *exec.Cmd, timeout time.Duration) ([]byte, error) {
	var buf bytes.Buffer

	cmd.Stdout = &buf
	cmd.Stderr = &buf

	doneCh := make(chan error)

	start := time.Now()

	err := cmd.Start()
	if Error(err) {
		return nil, err
	}

	Debug("process started pid: %d timeout: %v cmd: %s ...", cmd.Process.Pid, timeout, CmdToString(cmd))

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine(2))

		doneCh <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		Debug("process will be killed pid: %d timeout: %v cmd: %s time: %v", cmd.Process.Pid, timeout, CmdToString(cmd), time.Since(start))
		Error(cmd.Process.Kill())

		return nil, &ErrWatchdog{Msg: fmt.Sprintf("killed process pid: %d cmd: %s after: %v", cmd.Process.Pid, CmdToString(cmd), time.Since(start))}
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
		var output []byte

		switch exitcode {
		case 0:
			exitstate = "successfull"
			output = buf.Bytes()
		default:
			exitstate = "failed"
		}

		Debug("process %s! pid: %d exitcode: %d timeout: %v cmd: %s time: %s", exitstate, cmd.Process.Pid, exitcode, timeout, CmdToString(cmd), time.Since(start))
		Debug("%s", string(output))

		return output, err
	}
}

func NewWatchdogFunc(msg string, fn func() error, timeout time.Duration) error {
	doneCh := make(chan error)

	start := time.Now()

	var err error

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine(2))

		doneCh <- fn()
	}()

	select {
	case <-time.After(timeout):
		Debug("Watchdog function killed! time: %v", time.Since(start))

		return &ErrWatchdog{Msg: msg}
	case err = <-doneCh:
		exitstate := ""
		if err != nil {
			exitstate = "failed"
		} else {
			exitstate = "successfull"
		}

		Debug("Watchdog function %s! time: %s", exitstate, time.Since(start))
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
