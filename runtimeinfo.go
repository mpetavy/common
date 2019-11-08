package common

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const UNKNOWN = "unknwon"

type RuntimeInfo struct {
	Dir, Pack, File, Fn string
	Line                int
}

type SystemInfo struct {
	KernelName    string
	KernelVersion string
	KernelRelease string
	Platform      string
	MemTotal      string
	MemFree       string
}

type runner struct {
	cmd *exec.Cmd
	wg  *sync.WaitGroup

	err    error
	output string
}

func (this *runner) execute(cmd *exec.Cmd, timeout time.Duration, wg *sync.WaitGroup) {
	defer func() {
		if wg != nil {
			wg.Done()
		}
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

func (r RuntimeInfo) toString(asFilename bool) string {
	if asFilename {
		return fmt.Sprintf("%s-%s-%d-%s", r.Pack, r.File, r.Line, r.Fn)
	} else {
		return fmt.Sprintf("%s/%s:%d/%s", r.Pack, r.File, r.Line, r.Fn)
	}
}

func (r RuntimeInfo) String() string {
	return r.toString(false)
}

func (r RuntimeInfo) Filename() string {
	return r.toString(true)
}

func GetRuntimeInfo(pos int) RuntimeInfo {
	pc, _, _, ok := runtime.Caller(1 + pos)

	if !ok {
		return RuntimeInfo{UNKNOWN, UNKNOWN, UNKNOWN, UNKNOWN, 0}
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

	return RuntimeInfo{dir, pack, file, fn, line}
}

func readStringTable(txt string, separator string) []string {
	r := bufio.NewReader(strings.NewReader(txt))

	lines := make([]string, 2)
	splits := make([]string, 0)

	length := 0

	for i := 0; i < 2; i++ {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		lines[i] = line
		length = Max(length, len(lines[i]))
	}

	start := 0

	for col := 0; col < length; col++ {
		c := 0
		for l := 0; l < len(lines); l++ {
			if col >= len(lines[l]) || (col > 1) && lines[l][col-2:col] == separator {
				c++
			}
		}

		if c == len(lines) {
			split := strings.TrimSpace(lines[1][start:col])

			splits = append(splits, split)

			start = col
		}
	}

	return splits
}

func GetSystemInfo() (SystemInfo, error) {
	DebugFunc()

	si := SystemInfo{}

	if IsWindowsOS() {
		var wmicOSRunner runner

		wmicOSRunner.execute(exec.Command("wmic", "os"), time.Second, nil)

		splits := readStringTable(wmicOSRunner.output, "  ")

		si.MemFree = splits[20] + "+" + splits[22]
		mem0, err := strconv.Atoi(splits[20])
		if err == nil {
			mem1, err := strconv.Atoi(splits[22])

			if err == nil {
				mem := float64(mem0+mem1) / float64(1024*1024)

				si.MemFree = fmt.Sprintf("%f MB", mem)

			}
		}

		si.MemTotal = splits[60]
		mem0, err = strconv.Atoi(splits[60])
		if err == nil {
			mem := float64(mem0) / float64(1024*1024)

			si.MemTotal = fmt.Sprintf("%f MB", mem)
		}

		si.KernelName = splits[3]
		si.KernelVersion = splits[62]
		si.KernelRelease = splits[2]
		si.Platform = splits[38]

		return si, nil
	}

	var wg sync.WaitGroup

	wg.Add(5)

	var kernelNameRunner runner
	var kernelReleaseRunner runner
	var kernelVersionRunner runner
	var machineRunner runner

	go kernelNameRunner.execute(exec.Command("uname", "-s"), time.Second, &wg)
	go kernelReleaseRunner.execute(exec.Command("uname", "-r"), time.Second, &wg)
	go kernelVersionRunner.execute(exec.Command("uname", "-v"), time.Second, &wg)
	go machineRunner.execute(exec.Command("uname", "-m"), time.Second, &wg)
	go func(si *SystemInfo) {
		ba, err := ioutil.ReadFile("/proc/meminfo")
		if err != nil {
			return
		}

		scanner := bufio.NewScanner(bytes.NewBuffer(ba))
		for scanner.Scan() {
			txt := scanner.Text()

			splits := strings.Split(txt, ":")
			if len(splits) != 2 {
				continue
			}

			for i := 0; i < len(splits); i++ {
				splits[i] = strings.TrimSpace(splits[i])
			}

			if strings.HasPrefix(strings.ToLower(splits[0]), "memtotal") {
				si.MemTotal = splits[1]
			}
			if strings.HasPrefix(strings.ToLower(splits[0]), "memfree") {
				si.MemFree = splits[1]
			}
		}

		wg.Done()
	}(&si)

	wg.Wait()

	si.KernelName = kernelNameRunner.output
	si.KernelRelease = kernelReleaseRunner.output
	si.KernelVersion = kernelVersionRunner.output
	si.Platform = machineRunner.output

	return si, nil
}
