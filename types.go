package common

import (
	"fmt"
	"github.com/paulrosania/go-charset/charset"
	_ "github.com/paulrosania/go-charset/data"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
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
	firstRune := runes[0]
	lastRune := runes[len(runes)-1]

	if lastRune == unicode.ToUpper(lastRune) {
		return txt
	} else {
		return fmt.Sprintf("%s%s", string(unicode.ToUpper(firstRune)), string(runes[1:]))
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

	if err != nil {
		return "", nil
	}

	return string(b), nil
}

func ToUTF8(r io.Reader, cs string) ([]byte, error) {
	rcs, err := charset.NewReader(cs, r)
	if err != nil {
		return []byte{}, err
	}
	b, err := ioutil.ReadAll(rcs)

	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

func IndexOf(slice interface{}, search interface{}) (int, error) {
	if reflect.TypeOf(slice).Kind() != reflect.Slice {
		return -1, errors.WithStack(fmt.Errorf("not a slice: %v", slice))
	}

	sl := reflect.ValueOf(slice)

	if sl.Len() == 0 {
		return -1, nil
	}

	s := fmt.Sprintf("%v", search)

	for i := 0; i < sl.Len(); i++ {
		c := fmt.Sprintf("%v", sl.Index(i))
		if c == s {
			return i, nil
		}
	}

	return -1, nil
}

func ToStrings(slice interface{}) ([]string, error) {
	if reflect.TypeOf(slice).Kind() != reflect.Slice {
		return []string{}, errors.WithStack(fmt.Errorf("not a slice: %v", slice))
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

func ConvertToOSspecificLF(s string) string {
	if IsWindowsOS() {
		s = strings.ReplaceAll(s, "\r", "\r\n")
	}

	return s
}

func CountRunes(s string) int {
	return len([]rune(s))
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
	mask = strings.ReplaceAll(mask, ".", "\\.")
	mask = strings.ReplaceAll(mask, "*", ".*")
	mask = strings.ReplaceAll(mask, "?", ".")

	r, err := regexp.Compile("^" + mask + "$")
	if err != nil {
		return false, err
	}

	return r.MatchString(s), nil
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

func IterateStruct(data interface{}, function func(path string, fieldType reflect.StructField, fieldValue reflect.Value) error) error {
	return iterateStruct("", data, function)
}

func iterateStruct(path string, data interface{}, funcStructIterator func(path string, fieldType reflect.StructField, fieldValue reflect.Value) error) error {
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

	if funcStructIterator != nil {
		err := funcStructIterator(path, reflect.StructField{}, val)
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

		if funcStructIterator != nil {
			err := funcStructIterator(fieldPath, typ.Field(i), val.Field(i))
			if Error(err) {
				return err
			}
		}

		if val.Field(i).Kind() == reflect.Struct {
			err := iterateStruct(fieldPath, val.Field(i), funcStructIterator)
			if err != nil {
				return err
			}

			continue
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
		if strings.HasSuffix(txt, strings.ToLower(MEMORY_UNITS[i])) || strings.HasSuffix(txt, strings.ToLower(MEMORY_UNITS[i][:1])) {
			return int64(f * math.Pow(1024, float64(i))), nil
		}
	}

	return int64(f * math.Pow(1024, float64(0))), nil
}

func FormatMemory(mem int) string {
	neg := mem < 0

	f := math.Abs(float64(mem))

	var i int
	var d float64

	for i = len(MEMORY_UNITS) - 1; i >= 0; i-- {
		d = math.Pow(1024, float64(i))

		if f > d {
			break
		}
	}

	r := f / d

	if neg {
		r = r * -1
	}

	return fmt.Sprintf("%.3f %s", r, MEMORY_UNITS[Max(i, 0)])
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
