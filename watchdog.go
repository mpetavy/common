package common

import (
	"bytes"
	"fmt"
	"os/exec"
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
	DebugFunc("%s: %d msec...", CmdToString(cmd), timeout.Milliseconds())

	var buf bytes.Buffer

	cmd.Stdout = &buf
	cmd.Stderr = &buf

	doneCh := make(chan error, 1)

	start := time.Now()

	go func() {
		defer UnregisterGoRoutine(RegisterGoRoutine(2))

		err := cmd.Start()
		if Error(err) {
			doneCh <- err

			return
		}

		Debug("Watchdog process started pid: %d timeout: %v cmd: %s ...", cmd.Process.Pid, timeout, CmdToString(cmd))

		err = cmd.Wait()
		if Error(err) {
			doneCh <- err
		}

		close(doneCh)
	}()

	select {
	case <-time.After(timeout):
		Error(cmd.Process.Kill())

		return nil, &ErrWatchdog{Msg: fmt.Sprintf("Watchdog process is killed by timeout! pid: %d timeout: %v cmd: %s time: %v", cmd.Process.Pid, timeout, CmdToString(cmd), time.Since(start))}
	case err := <-doneCh:
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

		if exitcode == 0 {
			exitstate = "successfull"
			output = buf.Bytes()
		} else {
			exitstate = "failed"
		}

		msg := fmt.Sprintf("Watchdog process %s! pid: %d exitcode: %d timeout: %v cmd: %s time: %s error: %v", exitstate, cmd.Process.Pid, exitcode, timeout, CmdToString(cmd), time.Since(start), err)

		if exitcode == 0 {
			Debug(msg)
			Debug("%s", string(output))

			return output, nil
		} else {
			return nil, fmt.Errorf(msg)
		}
	}
}
