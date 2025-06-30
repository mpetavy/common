package common

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
)

const (
	indent = "    "
)

type StringTable struct {
	Cells     [][]string
	NoHeader  bool
	Alignment []int
}

func NewStringTable() *StringTable {
	return &StringTable{}
}

func writeString(w io.Writer, s string) error {
	_, err := w.Write([]byte(s))

	return err
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

		line.WriteString(fmt.Sprintf(fmt.Sprintf("%%-%ds", colLengths[x]), cols[x]))
	}

	txt := line.String()

	if markdown {
		txt = "| " + txt + " |"
	}

	txt += "\n"

	return txt
}

func (st *StringTable) Table() string {
	buf := bytes.Buffer{}

	Error(st.table(&buf, false))

	return string(buf.Bytes())
}

func (st *StringTable) TableToWriter(w io.Writer) error {
	err := st.table(w, false)
	if Error(err) {
		return err
	}

	return nil
}

func (st *StringTable) Markdown() string {
	buf := bytes.Buffer{}

	Error(st.table(&buf, true))

	return string(buf.Bytes())
}

func (st *StringTable) MarkdownToWriter(w io.Writer) error {
	err := st.table(w, true)
	if Error(err) {
		return err
	}

	return nil
}

func (st *StringTable) table(w io.Writer, markdown bool) error {
	colLengths := make([]int, 0)

	for y := 0; y < len(st.Cells); y++ {
		for len(colLengths) < len(st.Cells[y]) {
			colLengths = append(colLengths, 0)
		}

		for x := 0; x < len(st.Cells[y]); x++ {
			colLengths[x] = max(colLengths[x], len(st.Cells[y][x]))
		}
	}

	for y := 0; y < len(st.Cells); y++ {
		if y > 0 || !st.NoHeader {
			err := writeString(w, st.rower(markdown, st.Cells[y], colLengths, false))
			if Error(err) {
				return err
			}

			if y == 0 {
				sep := make([]string, len(st.Cells[0]))
				for x := 0; x < len(sep); x++ {
					s := fmt.Sprintf("%s", strings.Repeat("-", colLengths[x]))

					sep[x] = s
				}
				err := writeString(w, st.rower(markdown, sep, colLengths, true))
				if Error(err) {
					return err
				}
			}
		}
	}

	return nil
}

func (st *StringTable) html(w io.Writer) error {
	err := writeString(w, "<table>\n")
	if Error(err) {
		return err
	}

	for y := 0; y < len(st.Cells); y++ {
		if y == 0 {
			if !st.NoHeader {
				err := writeString(w, "\t<thead>\n")
				if Error(err) {
					return err
				}
			} else {
				err := writeString(w, "\t<tbody>\n")
				if Error(err) {
					return err
				}
			}
		}

		err := writeString(w, "\t<tr>\n")
		if Error(err) {
			return err
		}

		for x := 0; x < len(st.Cells[y]); x++ {
			err := writeString(w, "\t\t<td>")
			if Error(err) {
				return err
			}
			err = writeString(w, fmt.Sprintf("%v", st.Cells[y][x]))
			if Error(err) {
				return err
			}
			err = writeString(w, "</td>\n")
			if Error(err) {
				return err
			}
		}

		err = writeString(w, "\t</tr>\n")
		if Error(err) {
			return err
		}

		if y == 0 {
			if !st.NoHeader {
				err := writeString(w, "\t</thead>\n")
				if Error(err) {
					return err
				}
			} else {
				err := writeString(w, "\t</tbody>\n")
				if Error(err) {
					return err
				}
			}
		}
	}

	err = writeString(w, "</table>\n")
	if Error(err) {
		return err
	}

	return nil
}

func (st *StringTable) HTML() string {
	buf := bytes.Buffer{}

	Error(st.html(&buf))

	return string(buf.Bytes())
}

func (st *StringTable) HTMLToWriter(w io.Writer) error {
	err := st.html(w)
	if Error(err) {
		return err
	}

	return nil
}

func (st *StringTable) json(w io.Writer) error {
	err := writeString(w, "[\n")
	if Error(err) {
		return err
	}

	if len(st.Cells) > 1 {
		for y := 1; y < len(st.Cells); y++ {
			if y > 1 {
				err := writeString(w, ",\n")
				if Error(err) {
					return err
				}
			}

			err := writeString(w, fmt.Sprintf("%s{\n", indent))
			if Error(err) {
				return err
			}

			for x := 0; x < len(st.Cells[y]); x++ {
				if x > 0 {
					err := writeString(w, ",\n")
					if Error(err) {
						return err
					}
				}

				name, _ := json.Marshal(st.Cells[0][x])
				name = name[1 : len(name)-1]

				value, _ := json.Marshal(st.Cells[y][x])
				value = value[1 : len(value)-1]

				err := writeString(w, fmt.Sprintf("%s%s\"%v\": \"%v\"", indent, indent, string(name), string(value)))
				if Error(err) {
					return err
				}
			}

			err = writeString(w, fmt.Sprintf("\n"))
			if Error(err) {
				return err
			}

			err = writeString(w, fmt.Sprintf("%s}", indent))
			if Error(err) {
				return err
			}
		}
		err := writeString(w, "\n")
		if Error(err) {
			return err
		}
	}

	err = writeString(w, "]\n")
	if Error(err) {
		return err
	}

	return nil
}

func (st *StringTable) JSON() string {
	buf := bytes.Buffer{}

	Error(st.json(&buf))

	return string(buf.Bytes())
}

func (st *StringTable) JSONToWriter(w io.Writer) error {
	err := st.json(w)
	if Error(err) {
		return err
	}

	return nil
}

func (st *StringTable) csv(w io.Writer) error {
	for y := 0; y < len(st.Cells); y++ {
		if y == 0 && st.NoHeader {
			continue
		}

		for x := 0; x < len(st.Cells[y]); x++ {
			if x > 0 {
				err := writeString(w, ",")
				if Error(err) {
					return err
				}
			}
			value := st.Cells[y][x]

			value = strings.ReplaceAll(value, "\"", "\"\"")

			err := writeString(w, fmt.Sprintf("\"%v\"", value))
			if Error(err) {
				return err
			}
		}

		err := writeString(w, "\n")
		if Error(err) {
			return err
		}
	}

	return nil
}

func (st *StringTable) CSV() string {
	buf := bytes.Buffer{}

	Error(st.csv(&buf))

	return string(buf.Bytes())
}

func (st *StringTable) CSVToWriter(w io.Writer) error {
	err := st.csv(w)
	if Error(err) {
		return err
	}

	return nil
}

func (st *StringTable) Debug() {
	scanner := bufio.NewScanner(strings.NewReader(st.Table()))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		Debug(line)
	}
}
