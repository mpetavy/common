package common

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/paulrosania/go-charset/charset"
	_ "github.com/paulrosania/go-charset/data"
	"io"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	BIG5         = "big5"
	IBM437       = "ibm437"
	IBM850       = "ibm850"
	IBM866       = "ibm866"
	ISO_8859_1   = "iso-8859-1"
	ISO_8859_10  = "iso-8859-10"
	iso_8859_15  = "iso-8859-15"
	ISO_8859_2   = "iso-8859-2"
	ISO_8859_3   = "iso-8859-3"
	ISO_8859_4   = "iso-8859-4"
	ISO_8859_5   = "iso-8859-5"
	ISO_8859_6   = "iso-8859-6"
	ISO_8859_7   = "iso-8859-7"
	ISO_8859_8   = "iso-8859-8"
	ISO_8859_9   = "iso-8859-9"
	KOI8_R       = "koi8-r"
	US_ASCII     = "us-ascii"
	UTF_16       = "utf-16"
	UTF_16BE     = "utf-16be"
	UTF_16LE     = "utf-16le"
	UTF_8        = "utf-8"
	WINDOWS_1250 = "windows-1250"
	WINDOWS_1251 = "windows-1251"
	WINDOWS_1252 = "windows-1252"
)

var (
	MEMORY_UNITS = []string{"Bytes", "KB", "MB", "GB", "TB"}
)

// Trim4Path trims given path to be usefully as filename
func Trim4Path(path string) string {
	spath := []rune(path)
	for i, ch := range spath {
		if ch == ':' || ch == '\\' {
			spath[i] = '_'
		}
	}

	return string(spath)
}

// CompareIgnoreCase compares strings for equality ignoring case
func CompareIgnoreCase(s0 string, s1 string) bool {
	return strings.ToLower(s0) == strings.ToLower(s1)
}

// SplitWithQuotation splits a sequented string by spaces and respects quotation
func SplitWithQuotation(txt string) []string {
	var result []string
	var line string
	inQ := false

	for _, c := range txt {
		if c == '"' {
			inQ = !inQ
		} else if c == ' ' {
			if !inQ {
				if len(line) > 0 {
					result = append(result, line)
					line = ""
				} else {
					line += string(c)
				}
			} else {
				line += string(c)
			}
		} else {
			line += string(c)
		}
	}

	if len(line) > 0 {
		result = append(result, line)
	}

	return result
}

// Capitalize the first letter
func Capitalize(txt string) string {
	if len(txt) == 0 {
		return txt
	}

	runes := []rune(txt)

	allUpper := true
	for _, r := range runes {
		allUpper = unicode.IsUpper(r)

		if !allUpper {
			break
		}
	}

	if allUpper {
		return txt
	} else {
		return fmt.Sprintf("%s%s", string(unicode.ToUpper(runes[0])), string(runes[1:]))
	}
}

func SurroundWith(str []string, prefixSuffix string) []string {
	var result []string

	result = make([]string, len(str))

	for i, v := range str {
		result[i] = prefixSuffix + v + prefixSuffix
	}

	return result
}

func Shortener(s string, max int) string {
	if len(s) > max {
		s = s[:max-4] + "..."
	}

	return s
}

func DefaultEncoding() string {
	if IsWindowsOS() {
		return ISO_8859_1
	} else {
		return UTF_8
	}
}

func DefaultConsoleEncoding() string {
	if IsWindowsOS() {
		return IBM850
	} else {
		return UTF_8
	}
}

func ToUTF8String(s string, cs string) (string, error) {
	b, err := ToUTF8(strings.NewReader(s), cs)

	if Error(err) {
		return "", err
	}

	return string(b), nil
}

func ToUTF8(r io.Reader, cs string) ([]byte, error) {
	rcs, err := charset.NewReader(cs, r)
	if Error(err) {
		return []byte{}, err
	}
	b, err := io.ReadAll(rcs)

	if Error(err) {
		return []byte{}, err
	}
	return b, nil
}

func Contains(slice interface{}, search interface{}) bool {
	return IndexOf(slice, search) != -1
}

func IndexOf(slice interface{}, search interface{}) int {
	if reflect.TypeOf(slice).Kind() != reflect.Slice {
		panic(fmt.Errorf("not a slice: %v", slice))
	}

	sl := reflect.ValueOf(slice)

	if sl.Len() == 0 {
		return -1
	}

	s := fmt.Sprintf("%v", search)

	for i := 0; i < sl.Len(); i++ {
		c := fmt.Sprintf("%v", sl.Index(i))
		if c == s {
			return i
		}
	}

	return -1
}

func ToStrings(slice interface{}) ([]string, error) {
	if reflect.TypeOf(slice).Kind() != reflect.Slice {
		return []string{}, fmt.Errorf("not a slice: %v", slice)
	}

	sl := reflect.ValueOf(slice)

	if sl.Len() == 0 {
		return nil, nil
	}

	var result []string

	for i := 0; i < sl.Len(); i++ {
		result = append(result, fmt.Sprintf("%v", sl.Index(i).Interface()))
	}
	return result, nil
}

func Rune(s string, index int) (rune, error) {
	runes := []rune(s)

	if index < len(runes) {
		return runes[index], nil
	} else {
		return rune(' '), fmt.Errorf("invalid rune position: %d", index)
	}
}

func ContainsWildcard(s string) bool {
	return strings.ContainsAny(s, "*?")
}

func EqualWildcards(s, mask string) (bool, error) {
	if !ContainsWildcard(mask) {
		return strings.ToLower(s) == strings.ToLower(mask), nil
	}

	mask = strings.ReplaceAll(mask, ".", "\\.")
	mask = strings.ReplaceAll(mask, "*", ".*")
	mask = strings.ReplaceAll(mask, "?", ".")
	mask = "(?i)" + mask

	b, err := regexp.Match(mask, []byte(s))
	if Error(err) {
		return false, err
	}

	return b, err
}

func ReflectStructField(Iface interface{}, FieldName string) (*reflect.Value, error) {
	valueIface := reflect.ValueOf(Iface)

	// Check if the passed interface is a pointer
	if valueIface.Type().Kind() != reflect.Ptr {
		// Create a new type of Iface's Type, so we have a pointer to work with
		valueIface = reflect.New(reflect.TypeOf(Iface))
	}

	// 'dereference' with Elem() and get the field by name
	field := valueIface.Elem().FieldByName(FieldName)
	if !field.IsValid() {
		return nil, fmt.Errorf("Interface `%s` does not have the field `%s`", valueIface.Type(), FieldName)
	}

	return &field, nil
}

func ReflectStructMethod(Iface interface{}, MethodName string) (*reflect.Value, error) {
	valueIface := reflect.ValueOf(Iface)

	// Check if the passed interface is a pointer
	if valueIface.Type().Kind() != reflect.Ptr {
		// Create a new type of Iface, so we have a pointer to work with
		valueIface = reflect.New(reflect.TypeOf(Iface))
	}

	// Get the method by name
	method := valueIface.MethodByName(MethodName)
	if !method.IsValid() {
		return nil, fmt.Errorf("Couldn't find method `%s` in interface `%s`, is it Exported?", MethodName, valueIface.Type())
	}
	return &method, nil
}

func IterateStruct(data interface{}, fieldFunc func(path string, fieldType reflect.StructField, fieldValue reflect.Value) error) error {
	return iterateStruct("", data, fieldFunc)
}

func iterateStruct(path string, data interface{}, fieldFunc func(path string, fieldType reflect.StructField, fieldValue reflect.Value) error) error {
	DebugFunc()

	val, ok := data.(reflect.Value)
	if !ok {
		val = reflect.Indirect(reflect.ValueOf(data))
	}
	typ := val.Type()

	if typ.Kind() != reflect.Struct {
		return fmt.Errorf("not a struct: %s", typ.Name())
	}

	DebugFunc("struct: %s", typ.Name())

	if fieldFunc != nil {
		err := fieldFunc(path, reflect.StructField{}, val)
		if Error(err) {
			return err
		}
	}

	for i := 0; i < val.NumField(); i++ {
		fieldPath := typ.Field(i).Name
		if path != "" {
			fieldPath = strings.Join([]string{path, fieldPath}, "_")
		}

		DebugFunc("field #%d %s : fieldpath: %s type: %s",
			i,
			typ.Field(i).Name,
			fieldPath,
			val.Field(i).Type().Name())

		if val.Field(i).Kind() == reflect.Struct {
			err := fieldFunc(path, typ.Field(i), val)
			if Error(err) {
				return err
			}

			err = iterateStruct(fieldPath, val.Field(i), fieldFunc)
			if Error(err) {
				return err
			}

			continue
		}

		if val.Field(i).Kind() == reflect.Slice && val.Field(i).Type().Kind() == reflect.Struct {
			for j := 0; j < val.Field(i).Len(); j++ {
				sliceFieldPath := fmt.Sprintf("%s[%d]", fieldPath, j)
				sliceElement := val.Field(i).Index(j).Elem()

				if sliceElement.Kind() == reflect.Struct {
					err := iterateStruct(sliceFieldPath, sliceElement, fieldFunc)
					if Error(err) {
						return err
					}

					continue
				}

				if fieldFunc != nil {
					err := fieldFunc(sliceFieldPath, reflect.StructField{}, sliceElement)
					if Error(err) {
						return err
					}
				}
			}

			continue
		}

		if fieldFunc != nil {
			err := fieldFunc(fieldPath, typ.Field(i), val.Field(i))
			if Error(err) {
				return err
			}
		}
	}

	return nil
}

func FillString(txt string, length int, asPrefix bool, add string) string {
	for len(txt) < length {
		if asPrefix {
			txt = add + txt
		} else {
			txt = txt + add
		}
	}

	return txt[:length]
}

func ParseMemory(txt string) (int64, error) {
	txt = strings.ToLower(txt)

	f, err := ExtractNumber(txt)
	if Error(err) {
		return 0, err
	}

	for i := 0; i < len(MEMORY_UNITS); i++ {
		if strings.HasSuffix(txt, strings.ToLower(MEMORY_UNITS[i])) || (i > 0 && strings.HasSuffix(txt, strings.ToLower(MEMORY_UNITS[i][:1]))) {
			return int64(f * math.Pow(1024, float64(i))), nil
		}
	}

	return int64(f * math.Pow(1024, float64(0))), nil
}

func FormatMemory(bytes int64) string {
	neg := bytes < 0

	fbytes := math.Abs(float64(bytes))

	var i int
	var d float64

	for i = len(MEMORY_UNITS) - 1; i >= 0; i-- {
		d = math.Pow(1024, float64(i))

		if fbytes > d && (fbytes/d > 10) {
			break
		}
	}

	r := fbytes / d

	if neg {
		r = r * -1
	}

	return fmt.Sprintf("%.2f %s", r, MEMORY_UNITS[Max(i, 0)])
}

func ExtractNumber(txt string) (float64, error) {
	r := regexp.MustCompile("[\\d.-]*")
	s := r.FindString(txt)

	if s == "" {
		return -1, fmt.Errorf("cannot getNumber() from %s", txt)
	}

	f, err := strconv.ParseFloat(s, 64)
	if Error(err) {
		return -1, err
	}

	return f, nil
}

func SortStringsCaseInsensitive(strs []string) {
	sort.SliceStable(strs, func(i, j int) bool {
		return strings.ToUpper(strs[i]) < strings.ToUpper(strs[j])
	})
}

func Join(strs []string, sep string) string {
	sb := strings.Builder{}
	for i := 0; i < len(strs); i++ {
		if strs[i] != "" {
			if sb.Len() > 0 {
				sb.WriteString(sep)
			}
			sb.WriteString(strs[i])

		}
	}

	return sb.String()
}

func Clear(v interface{}) error {
	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		return fmt.Errorf("not a pointer")
	}

	p := reflect.ValueOf(v).Elem()
	p.Set(reflect.Zero(p.Type()))

	return nil
}

type separatorSplitFunc struct {
	prefix []byte
	suffix []byte

	remove bool
	fn     bufio.SplitFunc
}

func NewSplitFuncSeparator(prefix []byte, suffix []byte, remove bool) (bufio.SplitFunc, error) {
	if suffix == nil {
		return nil, fmt.Errorf("at least the suffix must be defined")
	}

	s := separatorSplitFunc{
		prefix: prefix,
		suffix: suffix,
		remove: remove,
	}

	return s.splitFunc, nil
}

func (s *separatorSplitFunc) splitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	indexPrefix := 0
	if s.prefix != nil {
		indexPrefix = bytes.Index(data, s.prefix)
	}
	indexSuffix := bytes.Index(data, s.suffix)

	if indexSuffix != -1 && (s.prefix == nil || (indexPrefix != -1 && indexPrefix < indexSuffix)) {
		deltaPrefix := 0
		deltaSuffix := 0

		if s.remove {
			deltaPrefix = len(s.prefix)
			deltaSuffix = len(s.suffix)
		}

		return indexSuffix + len(s.suffix), data[indexPrefix+deltaPrefix : indexSuffix+len(s.suffix)-deltaSuffix], nil
	}

	if atEOF {
		return 0, nil, io.EOF
	}

	return 0, nil, nil
}
