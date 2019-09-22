package common

import (
	"fmt"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const UNKNOWN = "unknwon"

type runtimeInfo struct {
	Dir, Pack, File, Fn string
	Line                int
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
