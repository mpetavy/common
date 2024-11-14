package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
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

func (st *StringTable) InsertCols(row int, txts ...interface{}) {
	cols := []any{}
	for _, txt := range txts {
		val := reflect.ValueOf(txt)
		if val.Type().Kind() == reflect.Slice || val.Type().Kind() == reflect.Array {
			for i := 0; i < val.Len(); i++ {
				cols = append(cols, fmt.Sprintf("%v", val.Index(i)))
			}

			continue

		}

		cols = append(cols, fmt.Sprintf("%v", txt))
	}

	st.Cells = slices.Insert(st.Cells, row, cols)
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

func (st *StringTable) Html() string {
	sb := strings.Builder{}
	sb.WriteString("<table>\n")

	for y := 0; y < len(st.Cells); y++ {
		if y == 0 {
			if !st.NoHeader {
				sb.WriteString("\t<thead>\n")
			} else {
				sb.WriteString("\t<tbody>\n")
			}
		}

		sb.WriteString("\t<tr>\n")

		for x := 0; x < len(st.Cells[y]); x++ {
			sb.WriteString("\t\t<td>")
			sb.WriteString(fmt.Sprintf("%v", st.Cells[y][x]))
			sb.WriteString("</td>\n")
		}

		sb.WriteString("\t</tr>\n")

		if y == 0 {
			if !st.NoHeader {
				sb.WriteString("\t</thead>\n")
			} else {
				sb.WriteString("\t</tbody>\n")
			}
		}
	}

	sb.WriteString("</table>\n")

	return sb.String()
}

func (st *StringTable) JSON(indent string) ([]byte, error) {
	buf := bytes.Buffer{}
	buf.WriteString("[\n")

	if len(st.Cells) > 1 {
		for y := 1; y < len(st.Cells); y++ {
			if y > 1 {
				buf.WriteString(",\n")
			}

			buf.WriteString(fmt.Sprintf("%s{\n", indent))

			for x := 0; x < len(st.Cells[y]); x++ {
				if x > 0 {
					buf.WriteString(",\n")
				}

				name, err := json.Marshal(st.Cells[0][x])
				if Error(err) {
					return nil, err
				}

				name = name[1 : len(name)-1]

				value, err := json.Marshal(st.Cells[y][x])
				if Error(err) {
					return nil, err
				}

				value = value[1 : len(value)-1]

				buf.WriteString(fmt.Sprintf("%s%s\"%v\": \"%v\"", indent, indent, string(name), string(value)))
			}
			buf.WriteString(fmt.Sprintf("\n"))
			buf.WriteString(fmt.Sprintf("%s}", indent))
		}
		buf.WriteString("\n")
	}

	buf.WriteString("]\n")

	return buf.Bytes(), nil
}

func (st *StringTable) CSV() string {
	sb := strings.Builder{}

	for y := 0; y < len(st.Cells); y++ {
		if y == 0 && st.NoHeader {
			continue
		}

		for x := 0; x < len(st.Cells[y]); x++ {
			if x > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(fmt.Sprintf("\"%v\"", st.Cells[y][x]))
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func (st *StringTable) Debug() {
	scanner := bufio.NewScanner(strings.NewReader(st.String()))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		Debug(line)
	}
}
