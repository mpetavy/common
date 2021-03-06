package common

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"
)

const UNKNOWN = "unknwon"

type RuntimeInfo struct {
	Dir, Pack, File, Fn, Stack string
	Line                       int
}

type SystemInfo struct {
	KernelName    string
	KernelVersion string
	KernelRelease string
	Platform      string
	MemTotal      string
	MemFree       string
}

type Runner struct {
	cmd     *exec.Cmd
	timeout time.Duration

	Err    error
	Output string
}

func NewRunner(cmd *exec.Cmd, timeout time.Duration) *Runner {
	return &Runner{
		cmd:     cmd,
		timeout: timeout,
	}
}

func (this *Runner) execute(wg *sync.WaitGroup) error {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	this.cmd.Stdout = &stdout
	this.cmd.Stderr = &stderr

	this.Err = WatchdogCmd(this.cmd, this.timeout)
	if Error(this.Err) {
		return this.Err
	}

	this.Output = string(stdout.Bytes())

	return nil
}

func MultiRunner(runners []Runner) error {
	chErr := ChannelError{}

	wg := sync.WaitGroup{}

	for i := range runners {
		go func(r *Runner) {
			err := r.execute(&wg)
			if err != nil {
				chErr.Add(err)
			}
		}(&runners[i])
	}

	wg.Wait()

	return chErr.Get()
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

	scanner := bufio.NewScanner(strings.NewReader(string(debug.Stack())))
	scanner.Split(ScanLinesWithLF)

	for i := 0; i < 1+((2+pos)*2); i++ {
		scanner.Scan()
	}

	stack := ""
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "\t") {
			stack += line
		}
	}

	if !ok {
		return RuntimeInfo{UNKNOWN, UNKNOWN, UNKNOWN, UNKNOWN, UNKNOWN, 0}
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

	return RuntimeInfo{dir, pack, file, fn, stack, line}
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

func GetSystemInfo() (*SystemInfo, error) {
	DebugFunc()

	si := &SystemInfo{}

	if IsWindowsOS() {
		r := NewRunner(exec.Command("wmic", "os"), time.Second*5)

		err := r.execute(nil)
		if Error(err) {
			return nil, err
		}

		splits := readStringTable(r.Output, "  ")

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

	kernelNameRunner := NewRunner(exec.Command("uname", "-s"), time.Second)
	kernelReleaseRunner := NewRunner(exec.Command("uname", "-r"), time.Second)
	kernelVersionRunner := NewRunner(exec.Command("uname", "-v"), time.Second)
	machineRunner := NewRunner(exec.Command("uname", "-m"), time.Second)

	go func() {
		Error(kernelNameRunner.execute(&wg))
	}()
	go func() {
		Error(kernelReleaseRunner.execute(&wg))
	}()
	go func() {
		Error(kernelVersionRunner.execute(&wg))
	}()
	go func() {
		Error(machineRunner.execute(&wg))
	}()
	go func(si *SystemInfo) {
		defer wg.Done()

		ba, err := os.ReadFile("/proc/meminfo")
		if Error(err) {
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
	}(si)

	wg.Wait()

	si.KernelName = strings.TrimSpace(kernelNameRunner.Output)
	si.KernelRelease = strings.TrimSpace(kernelReleaseRunner.Output)
	si.KernelVersion = strings.TrimSpace(kernelVersionRunner.Output)
	si.Platform = strings.TrimSpace(machineRunner.Output)

	DebugFunc("result: %v", *si)

	return si, nil
}
