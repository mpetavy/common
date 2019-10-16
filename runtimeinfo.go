package common

import (
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const UNKNOWN = "unknwon"

type runtimeInfo struct {
	Dir, Pack, File, Fn string
	Line                int
}

type systemInfo struct {
	KernelName    string
	KernelVersion string
	KernelRelease string
	Machine       string
}

type runner struct {
	cmd *exec.Cmd
	wg  *sync.WaitGroup

	err    error
	output string
}

var (
	si *systemInfo
)

func (this *runner) execute(cmd *exec.Cmd, timeout time.Duration, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	this.err = Watchdog(cmd, timeout)
	if Error(this.err) {
		return
	}

	this.output = string(stdout.Bytes())
}

func (r runtimeInfo) toString(asFilename bool) string {
	if asFilename {
		return fmt.Sprintf("%s-%s-%d-%s", r.Pack, r.File, r.Line, r.Fn)
	} else {
		return fmt.Sprintf("%s/%s:%d/%s", r.Pack, r.File, r.Line, r.Fn)
	}
}

func (r runtimeInfo) String() string {
	return r.toString(false)
}

func (r runtimeInfo) Filename() string {
	return r.toString(true)
}

func RuntimeInfo(pos int) runtimeInfo {
	pc, _, _, ok := runtime.Caller(1 + pos)

	if !ok {
		return runtimeInfo{UNKNOWN, UNKNOWN, UNKNOWN, UNKNOWN, 0}
	}

	f := runtime.FuncForPC(pc)

	fn := f.Name()
	fn = fn[strings.LastIndex(fn, ".")+1:]

	file, line := f.FileLine(pc)

	dir := filepath.Base(filepath.Dir(file))

	file = path.Base(file)

	pack := runtime.FuncForPC(pc).Name()
	pack = pack[strings.LastIndex(pack, "/")+1:]
	pack = pack[0:strings.Index(pack, ".")]

	return runtimeInfo{dir, pack, file, fn, line}
}

func removeApostroph(txt string) string {
	return txt[1 : len(txt)-1]
}

func SystemInfo() (*systemInfo, error) {
	DebugFunc()

	if si != nil {
		return si, nil
	}

	si = &systemInfo{}

	if IsWindowsOS() {
		cmd := exec.Command("systeminfo", "/fo", "csv", "/nh")

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := Watchdog(cmd, time.Millisecond*time.Duration(time.Second))
		if !Error(err) {
			splits := strings.Split(string(stdout.Bytes()), ",")

			si.KernelName = removeApostroph(splits[1])
			si.KernelRelease = removeApostroph(splits[5])
			si.KernelVersion = removeApostroph(splits[2][:strings.Index(splits[2], " ")])
			si.Machine = removeApostroph(splits[15])
		}

		return si, nil
	}

	var wg sync.WaitGroup

	wg.Add(4)

	var kernelNameRunner runner
	var kernelReleaseRunner runner
	var kernelVersionRunner runner
	var machineRunner runner

	go kernelNameRunner.execute(exec.Command("uname", "-s"), time.Second, &wg)
	go kernelReleaseRunner.execute(exec.Command("uname", "-r"), time.Second, &wg)
	go kernelVersionRunner.execute(exec.Command("uname", "-v"), time.Second, &wg)
	go machineRunner.execute(exec.Command("uname", "-m"), time.Second, &wg)

	wg.Wait()

	si.KernelName = kernelNameRunner.output
	si.KernelRelease = kernelReleaseRunner.output
	si.KernelVersion = kernelVersionRunner.output
	si.Machine = machineRunner.output

	return si, nil
}
