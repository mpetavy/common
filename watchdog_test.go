package common

import (
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestWatchdog(t *testing.T) {
	var cmd *exec.Cmd

	if IsWindows() {
		cmd = exec.Command("ping", "-t", "localhost")
	} else {
		cmd = exec.Command("ping", "localhost")
	}

	_, err := NewWatchdogCmd(cmd, time.Second*3)

	require.True(t, IsErrTimeout(err))
	require.True(t, os.IsTimeout(err))
}
