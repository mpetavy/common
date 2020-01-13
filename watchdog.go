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
	flagTimeout *int
)

func init() {
	flagTimeout = flag.Int("watchdog.timeout", 0, "watchdog timeout")
}

func (e *ErrWatchdog) Error() string {
	return fmt.Sprintf("watchdog killed process pid: %d cmd: %s after: %v", e.Cmd.Process.Pid, ToString(*e.Cmd), time.Since(e.Start))
}

func Watchdog(cmd *exec.Cmd, timeout time.Duration) error {
	if MsecToDuration(*flagTimeout) > timeout {
		timeout = MsecToDuration(*flagTimeout)
	}

	doneCh := make(chan error)

	start := time.Now()

	err := cmd.Start()
	if err != nil {
		return err
	}

	Debug("Watchdog observe process pid: %d timeout: %v cmd: %s ...", cmd.Process.Pid, timeout, ToString(*cmd))

	go func() {
		doneCh <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		Debug("Watchdog: process killed! pid: %d timeout: %v cmd: %s time: %s", cmd.Process.Pid, timeout, ToString(*cmd), time.Since(start))
		Error(cmd.Process.Kill())

		return &ErrWatchdog{cmd.Process.Pid, start, cmd}
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
		Debug("Watchdog: process %s! pid: %d exitcode: %d timeout: %v cmd: %s time: %s", exitstate, cmd.Process.Pid, exitcode, timeout, ToString(*cmd), time.Since(start))
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
