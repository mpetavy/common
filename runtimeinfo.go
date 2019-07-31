package common

import (
	"fmt"
	"path"
	"runtime"
	"strings"
)

const UNKNOWN = "unknwon"

type runtimeInfo struct {
	Pack, File, Fn string
	Line           int
}

func (r runtimeInfo) String(asFilename bool) string {
	if asFilename {
		return fmt.Sprintf("%s-%s-%s-%d", r.Pack, r.File, r.Fn, r.Line)
	} else {
		return fmt.Sprintf("%s/%s/%s:%d", r.Pack, r.File, r.Fn, r.Line)
	}
}

func RuntimeInfo(pos int) runtimeInfo {
	pc, _, _, ok := runtime.Caller(1 + pos)

	if !ok {
		return runtimeInfo{UNKNOWN, UNKNOWN, UNKNOWN, 0}
	}

	f := runtime.FuncForPC(pc)

	fn := f.Name()
	fn = fn[strings.LastIndex(fn, ".")+1:]

	file, line := f.FileLine(pc)
	file = path.Base(file)

	pack := runtime.FuncForPC(pc).Name()
	pack = pack[strings.LastIndex(pack, "/")+1:]
	pack = pack[0:strings.Index(pack, ".")]

	return runtimeInfo{pack, file, fn, line}
}
