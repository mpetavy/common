package common

import (
	"fmt"
	"reflect"
	"strings"
)

type StringTable struct {
	cols     [][]interface{}
	NoHeader bool
	Markdown bool
}

func NewStringTable() *StringTable {
	return &StringTable{}
}

func (st *StringTable) Clear() {
	st.cols = nil
}

func (st *StringTable) AddRow() {
	st.cols = append(st.cols, make([]interface{}, 0))
}

func (st *StringTable) AddCol(txt interface{}) {
	y := len(st.cols) - 1
	st.cols[y] = append(st.cols[y], fmt.Sprintf("%v", txt))
}

func (st *StringTable) AddCols(txts ...interface{}) {
	st.AddRow()
	for _, txt := range txts {
		val := reflect.ValueOf(txt)
		if val.Type().Kind() == reflect.Slice || val.Type().Kind() == reflect.Array {
			for i := 0; i < val.Len(); i++ {
				st.AddCol(val.Index(i))
			}

			continue

		}

		st.AddCol(txt)
	}
}

func (st *StringTable) rower(cols []interface{}, colLengths []int, cross bool) string {
	line := strings.Builder{}

	for x := 0; x < len(cols); x++ {
		format := fmt.Sprintf("%%-%dv", colLengths[x])
		if line.Len() > 0 {
			if cross {
				if st.Markdown {
					line.WriteString(" | ")
				} else {
					line.WriteString("-+-")
				}
			} else {
				line.WriteString(" | ")
			}
		}
		line.WriteString(fmt.Sprintf(format, cols[x]))
	}

	txt := line.String()

	if st.Markdown {
		txt = "| " + txt + " |"
	}

	txt += "\n"

	return txt
}

func (st *StringTable) String() string {
	colLengths := make([]int, 0)

	for y := 0; y < len(st.cols); y++ {
		for len(colLengths) < len(st.cols[y]) {
			colLengths = append(colLengths, 0)
		}

		for x := 0; x < len(st.cols[y]); x++ {
			colLengths[x] = max(colLengths[x], len(fmt.Sprintf("%v", st.cols[y][x])))
		}
	}

	sb := strings.Builder{}

	for y := 0; y < len(st.cols); y++ {
		sb.WriteString(st.rower(st.cols[y], colLengths, false))

		if y == 0 && !st.NoHeader {
			sep := make([]interface{}, len(st.cols[0]))
			for x := 0; x < len(sep); x++ {
				sep[x] = strings.Repeat("-", colLengths[x])
			}
			sb.WriteString(st.rower(sep, colLengths, true))
		}
	}

	return sb.String()
}
