package common

import (
	"fmt"
	"path"
	"runtime"
	"strings"
)

const UNKNOWN = "unknwon"

type ri struct {
	Pack, File, Fn string
	Line           int
}

func (r ri) String(asFilename bool) string {
	if asFilename {
		return fmt.Sprintf("%s-%s-%s-%d", r.Pack, r.File, r.Fn, r.Line)
	} else {
		return fmt.Sprintf("%s/%s.%s:%d", r.Pack, r.File, r.Fn, r.Line)
	}
}

func RuntimeInfo(pos int) ri {
	pc, _, _, ok := runtime.Caller(1 + pos)

	if !ok {
		return ri{UNKNOWN, UNKNOWN, UNKNOWN, 0}
	}

	f := runtime.FuncForPC(pc)

	fn := f.Name()
	fn = fn[strings.LastIndex(fn, ".")+1:]

	file, line := f.FileLine(pc)
	file = path.Base(file)

	pack := runtime.FuncForPC(pc).Name()
	pack = pack[strings.LastIndex(pack, "/")+1:]
	pack = pack[0:strings.Index(pack, ".")]

	return ri{pack, file, fn, line}
}
