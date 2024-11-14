package common

import (
	"bufio"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

type StringTable struct {
	Cells    [][]string
	NoHeader bool
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
	st.Cells = append(st.Cells, make([]string, 0))
}

func (st *StringTable) AddCol(txt string) {
	y := len(st.Cells) - 1
	st.Cells[y] = append(st.Cells[y], txt)
}

func (st *StringTable) AddCols(txts ...string) {
	st.AddRow()
	for _, txt := range txts {
		val := reflect.ValueOf(txt)
		if val.Type().Kind() == reflect.Slice || val.Type().Kind() == reflect.Array {
			for i := 0; i < val.Len(); i++ {
				st.AddCol(fmt.Sprintf("%v", val.Index(i)))
			}

			continue
		}

		st.AddCol(txt)
	}
}

func (st *StringTable) InsertCols(row int, txts ...string) {
	cols := []string{}
	for _, txt := range txts {
		val := reflect.ValueOf(txt)
		if val.Type().Kind() == reflect.Slice || val.Type().Kind() == reflect.Array {
			for i := 0; i < val.Len(); i++ {
				cols = append(cols, fmt.Sprintf("%v", val.Index(i)))
			}

			continue

		}

		cols = append(cols, txt)
	}

	st.Cells = slices.Insert(st.Cells, row, cols)
}

func (st *StringTable) rower(markdown bool, cols []string, colLengths []int, cross bool) string {
	line := strings.Builder{}

	for x := 0; x < len(cols); x++ {
		if line.Len() > 0 {
			if cross {
				if markdown {
					line.WriteString(" | ")
				} else {
					line.WriteString("-+-")
				}
			} else {
				line.WriteString(" | ")
			}
		}

		line.WriteString(cols[x] + strings.Repeat(" ", colLengths[x]-len(cols[x])))
	}

	txt := line.String()

	if markdown {
		txt = "| " + txt + " |"
	}

	txt += "\n"

	return txt
}

func (st *StringTable) Table() string {
	return st.table(false)
}

func (st *StringTable) Markdown() string {

	return st.table(true)
}

func (st *StringTable) table(markdown bool) string {
	colLengths := make([]int, 0)

	for y := 0; y < len(st.Cells); y++ {
		for len(colLengths) < len(st.Cells[y]) {
			colLengths = append(colLengths, 0)
		}

		for x := 0; x < len(st.Cells[y]); x++ {
			colLengths[x] = max(colLengths[x], len(st.Cells[y][x]))
		}
	}

	sb := strings.Builder{}

	for y := 0; y < len(st.Cells); y++ {
		sb.WriteString(st.rower(markdown, st.Cells[y], colLengths, false))

		if y == 0 && !st.NoHeader {
			sep := make([]string, len(st.Cells[0]))
			for x := 0; x < len(sep); x++ {
				s := fmt.Sprintf("%s", strings.Repeat("-", colLengths[x]))

				sep[x] = s
			}
			sb.WriteString(st.rower(markdown, sep, colLengths, true))
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

func (st *StringTable) JSON(indent string) string {
	sb := strings.Builder{}
	sb.WriteString("[\n")

	if len(st.Cells) > 1 {
		for y := 1; y < len(st.Cells); y++ {
			if y > 1 {
				sb.WriteString(",\n")
			}

			sb.WriteString(fmt.Sprintf("%s{\n", indent))

			for x := 0; x < len(st.Cells[y]); x++ {
				if x > 0 {
					sb.WriteString(",\n")
				}

				name, _ := json.Marshal(st.Cells[0][x])
				name = name[1 : len(name)-1]

				value, _ := json.Marshal(st.Cells[y][x])
				value = value[1 : len(value)-1]

				sb.WriteString(fmt.Sprintf("%s%s\"%v\": \"%v\"", indent, indent, string(name), string(value)))
			}
			sb.WriteString(fmt.Sprintf("\n"))
			sb.WriteString(fmt.Sprintf("%s}", indent))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("]\n")

	return sb.String()
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
	scanner := bufio.NewScanner(strings.NewReader(st.Table()))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		Debug(line)
	}
}
