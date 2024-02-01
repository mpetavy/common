package common

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	NONE = iota
	LEFT
	CENTER
	RIGHT
)

type StringTable struct {
	Cells    [][]interface{}
	NoHeader bool
	Markdown bool
	Aligment []int
}

func NewStringTable() *StringTable {
	return &StringTable{}
}

func (st *StringTable) Clear() {
	st.Cells = nil
}

func (st *StringTable) Rows() int {
	return len(st.Cells)
}

func (st *StringTable) AddRow() {
	st.Cells = append(st.Cells, make([]interface{}, 0))
}

func (st *StringTable) AddCol(txt interface{}) {
	y := len(st.Cells) - 1
	st.Cells[y] = append(st.Cells[y], fmt.Sprintf("%v", txt))
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

	for y := 0; y < len(st.Cells); y++ {
		for len(colLengths) < len(st.Cells[y]) {
			colLengths = append(colLengths, 0)
		}

		for x := 0; x < len(st.Cells[y]); x++ {
			colLengths[x] = max(colLengths[x], len(fmt.Sprintf("%v", st.Cells[y][x])))
		}
	}

	sb := strings.Builder{}

	for y := 0; y < len(st.Cells); y++ {
		sb.WriteString(st.rower(st.Cells[y], colLengths, false))

		if y == 0 && !st.NoHeader {
			sep := make([]interface{}, len(st.Cells[0]))
			for x := 0; x < len(sep); x++ {
				align := 0
				if x < len(st.Aligment) {
					align = st.Aligment[x]
				}

				s := ""

				switch align {
				case NONE:
					s = fmt.Sprintf("%s", strings.Repeat("-", colLengths[x]))
				case LEFT:
					s = fmt.Sprintf(":%s", strings.Repeat("-", colLengths[x]-1))
				case CENTER:
					s = fmt.Sprintf(":%s:", strings.Repeat("-", colLengths[x]-2))
				case RIGHT:
					s = fmt.Sprintf("%s:", strings.Repeat("-", colLengths[x]-1))
				}

				sep[x] = s
			}
			sb.WriteString(st.rower(sep, colLengths, true))
		}
	}

	return sb.String()
}
