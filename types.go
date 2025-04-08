package common

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/paulrosania/go-charset/charset"
	_ "github.com/paulrosania/go-charset/data"
	"github.com/saintfish/chardet"
	"golang.org/x/exp/constraints"
	"io"
	"math"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
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

	WindowsRestrictedFilenames = []string{"CON", "PRN", "AUX", "NUL", " COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
	SecretTags                 = []string{"username", "password", "token", "pwd", "credential", "subscription", "private", "accesskey", "secret", "endpoint", "secretkey", "authorization"}
)

// Trim4Path trims given path to be usefull as filename
func Trim4Path(path string) string {
	var spath string

	for _, ch := range []rune(path) {
		if strings.ContainsRune("\\ ", ch) {
			ch = rune('-')
		}
		if strings.ContainsRune("<>:.\"/|?*", ch) {
			ch = rune('_')
		}

		spath += string(ch)
	}

	path = string(spath)

	if IsWindows() {
		for _, reserved := range WindowsRestrictedFilenames {
			if path == reserved {
				path = path + "$"

				break
			}
		}
	}

	return path
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

func DefaultEncoding() string {
	if IsWindows() {
		return ISO_8859_1
	} else {
		return UTF_8
	}
}

func DefaultConsoleEncoding() string {
	if IsWindows() {
		return IBM850
	} else {
		return UTF_8
	}
}

func ToUTF8(ba []byte, inputEncoding string) ([]byte, error) {
	if inputEncoding == "" {
		detector := chardet.NewTextDetector()

		result, err := detector.DetectBest(ba)
		if !WarnError(err) {
			inputEncoding = result.Charset
		}
	}

	if strings.ToUpper(inputEncoding) == "UTF-8" {
		return ba, nil
	}

	rcs, err := charset.NewReader(inputEncoding, bytes.NewReader(ba))
	if Error(err) {
		return []byte{}, err
	}

	b, err := io.ReadAll(rcs)
	if Error(err) {
		return nil, err
	}

	return b, nil
}

func Contains(slice interface{}, search interface{}) bool {
	return IndexOf(slice, search) != -1
}

func IndexOf(slice interface{}, search interface{}) int {
	if reflect.TypeOf(slice).Kind() != reflect.Slice {
		Panic(fmt.Errorf("not a slice: %v", slice))
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

func EqualsWildcard(s, mask string) (bool, error) {
	if !ContainsWildcard(mask) {
		return strings.ToLower(s) == strings.ToLower(mask), nil
	}

	mask = strings.ReplaceAll(mask, ".", "\\.")
	mask = strings.ReplaceAll(mask, "*", ".*")
	mask = strings.ReplaceAll(mask, "?", ".")
	mask = "(?i)^" + mask + "$"

	b, err := regexp.Match(mask, []byte(s))
	if Error(err) {
		return false, err
	}

	return b, err
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

func FormatMemory(bytes uint64) string {
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

	return fmt.Sprintf("%.2f %s", r, MEMORY_UNITS[max(i, 0)])
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

func NewSeparatorSplitFunc(prefix []byte, suffix []byte, remove bool) (bufio.SplitFunc, error) {
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

		advance := indexSuffix + len(s.suffix)
		var err error

		if len(data) == advance && atEOF {
			err = bufio.ErrFinalToken
		}

		return advance, data[indexPrefix+deltaPrefix : indexSuffix+len(s.suffix)-deltaSuffix], err
	}

	if atEOF {
		return 0, nil, io.EOF
	}

	return 0, nil, nil
}

func Compare[T constraints.Ordered](a, b T) int {
	switch {
	case a < b:
		return -1
	case b > b:
		return 1
	default:
		return 0
	}
}

func ReverseSlice[T any](original []T) []T {
	reversed := make([]T, len(original))
	for i := 0; i < len(original); i++ {
		reversed[i] = original[len(original)-i-1]
	}

	return reversed
}

func Split(s string, sep string) []string {
	if s == "" {
		return []string{}
	}

	return strings.Split(s, sep)
}

func SplitCmdline(cmdline string) []string {
	var cmds []string

	var inCmd rune
	var cmd []rune

	addCmd := func() {
		if cmd != nil {
			cmds = append(cmds, string(cmd))
			cmd = nil
		}
	}

	scanCmd := func(ch rune) {
		switch inCmd {
		case 0:
			inCmd = ch
		case ch:
			inCmd = 0
		}
	}

	for _, ch := range cmdline {
		switch ch {
		case '"':
			scanCmd(ch)
		case '\'':
			scanCmd(ch)
		case ' ':
			if inCmd != 0 {
				cmd = append(cmd, ch)
			} else {
				addCmd()
			}
		default:
			cmd = append(cmd, ch)
		}
	}

	addCmd()

	return cmds
}

func PrintBytes(ba []byte, breakOnLineEndings bool) string {
	if ba == nil {
		return ""
	}

	sb := strings.Builder{}

	for _, r := range []rune(string(ba)) {
		if strconv.IsPrint(r) {
			sb.WriteString(string(r))
		} else {
			sb.WriteString("\\x")
			sb.WriteString(hex.EncodeToString([]byte(string(r))))
		}
	}

	str := sb.String()

	if breakOnLineEndings {
		endings := []string{
			"\\x0d\\x0a",
			"\\x0d",
			"\\x0a",
		}

		for _, ending := range endings {
			if strings.Index(str, ending) != -1 {
				str = strings.ReplaceAll(str, ending, ending+"\n")

				break
			}
		}
	}

	return str
}

func CmdToString(cmd *exec.Cmd) string {
	s := SurroundWith(cmd.Args, "\"")

	return strings.Join(s, " ")
}

func validateJson(p any, m map[string]any, ignoreUnknownFields bool) error {
	v, ok := p.(reflect.Value)
	if !ok {
		v = reflect.Indirect(reflect.ValueOf(p))
	}
	t := v.Type()

	var missing []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous || !field.IsExported() {
			continue
		}

		propertyName := field.Tag.Get("json")
		if propertyName == "" {
			propertyName = field.Name
		}

		if propertyName == "-" {
			continue
		}

		mp, ok := m[propertyName]
		delete(m, propertyName)
		if !ok {
			missing = append(missing, propertyName)
			continue
		}

		if field.Type.Kind() == reflect.Struct {
			props, ok := mp.(map[string]any)
			if !ok {
				return fmt.Errorf("properties map[string]any expected: %s", propertyName)
			}

			err := validateJson(v.FieldByName(field.Name), props, ignoreUnknownFields)
			if Error(err) {
				return err
			}
		}
	}

	if len(missing) > 0 {
		return errors.New("missing fields: " + strings.Join(missing, ", "))
	}

	if len(m) > 0 && !ignoreUnknownFields {
		extra := make([]string, 0, len(m))
		for field := range m {
			extra = append(extra, field)
		}

		return errors.New("unknown fields: " + strings.Join(extra, ", "))
	}

	return nil
}

func ValidateJson(p any, ba []byte, ignoreUnknownFields bool) error {
	if !json.Valid(ba) {
		return errors.New("invalid JSON data")
	}

	var m map[string]interface{}
	err := json.Unmarshal(ba, &m)
	if Error(err) {
		return err
	}

	return validateJson(p, m, ignoreUnknownFields)
}

func IndexNth(str string, substr string, count int) int {
	if count < 1 {
		return -1
	}

	i := 0
	for {
		if i+len(substr) > len(str) {
			return -1
		}

		if strings.HasPrefix(str[i:], substr) {
			count--

			if count == 0 {
				return i
			}
		}

		i++
	}
}

func Min[T constraints.Ordered](values ...T) T {
	var v T

	initialized := false

	for _, value := range values {
		if !initialized || value < v {
			initialized = true

			v = value
		}
	}

	return v
}

func Max[T constraints.Ordered](values ...T) T {
	var v T

	initialized := false

	for _, value := range values {
		if !initialized || value > v {
			initialized = true

			v = value
		}
	}

	return v
}

func GetStructValue(obj any, field string) (reflect.Value, error) {
	ref := reflect.ValueOf(obj)

	// if its a pointer, resolve its value
	if ref.Kind() == reflect.Ptr {
		ref = reflect.Indirect(ref)
	}

	if ref.Kind() == reflect.Interface {
		ref = ref.Elem()
	}

	// should double check we now have a struct (could still be anything)
	if ref.Kind() != reflect.Struct {
		return reflect.ValueOf(""), fmt.Errorf("not a struct type: %T", ref.Kind())
	}

	prop := ref.FieldByName(field)

	return prop, nil
}

func SetStructValue(obj any, field string, value any) error {
	ref := reflect.ValueOf(obj)

	// if its a pointer, resolve its value
	if ref.Kind() == reflect.Ptr {
		ref = reflect.Indirect(ref)
	}

	if ref.Kind() == reflect.Interface {
		ref = ref.Elem()
	}

	// should double check we now have a struct (could still be anything)
	if ref.Kind() != reflect.Struct {
		return fmt.Errorf("not a struct type: %T", ref.Kind())
	}

	prop := ref.FieldByName(field)
	prop.Set(reflect.ValueOf(value))

	return nil
}

type DurationJSON struct {
	time.Duration
}

func (d *DurationJSON) UnmarshalJSON(b []byte) (err error) {
	if b[0] == '"' {
		sd := string(b[1 : len(b)-1])
		d.Duration, err = time.ParseDuration(sd)
		return
	}

	var id int64
	id, err = json.Number(string(b)).Int64()
	d.Duration = time.Duration(id)

	return
}

func (d DurationJSON) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

func HideSecretFlags(name string, value string) string {
	if IsHashedValue(value) {
		return value
	}

	name = strings.ToLower(name)

	for _, hit := range SecretTags {
		if strings.Contains(name, hit) || strings.Contains(value, hit) {
			return strings.Repeat("X", Min(5, len(value))) + "..."
		}
	}

	return value
}

func HideSecrets(str string) string {
	var sb strings.Builder

	scanner := bufio.NewScanner(strings.NewReader(str))
	for scanner.Scan() {
		line := scanner.Text()

		for _, hit := range SecretTags {
			lowerLine := strings.ToLower(line)

			IgnoreError(Catch(func() error {
				p := strings.Index(lowerLine, hit)
				if p == -1 {
					return nil
				}

				p += len(hit)

				skip := strings.Index(lowerLine[p:], ":")
				if skip == -1 {
					skip = strings.Index(lowerLine[p:], "=")
				}
				if skip == -1 {
					skip = 0
				} else {
					skip += 1
				}

				p += skip

				line = line[:p] + strings.Repeat("X", 5) + "..."

				return nil
			}))
		}

		sb.WriteString(line + "\n")
	}

	return sb.String()
}

func IsEqualType(a any, b any) bool {
	return reflect.TypeOf(a) == reflect.TypeOf(b)
}

func CapString(txt string, max int) string {
	if len(txt) <= max {
		return txt
	}

	return txt[:max-3] + "..."
}

func ScrambleString(txt string) string {
	sb := strings.Builder{}
	for i := len(txt) - 1; i >= 0; i-- {
		sb.WriteByte(txt[i])
	}

	txt = sb.String()

	txt = txt[len(txt)/2:] + txt[:len(txt)/2]

	return sb.String()
}

func SortedKeys[K constraints.Ordered, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))

	for _key := range m {
		keys = append(keys, _key)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	return keys
}

func IsWhitespace(r rune) bool {
	return r == '\n' || r == '\r' || r == '\t'
}

func IsTextContent(ba []byte) (bool, error) {
	const maxRunes = 1000

	reader := bufio.NewReader(bytes.NewReader(ba))
	nonPrintable := 0
	total := 0

	for total < maxRunes {
		r, _, err := reader.ReadRune()
		if err != nil {
			break // EOF or error
		}

		if !unicode.IsPrint(r) && !IsWhitespace(r) {
			nonPrintable++
		}

		total++
	}

	if total == 0 {
		return true, nil // empty file = text
	}

	return nonPrintable == 0, nil
}

func IsTextFile(filename string) (bool, error) {
	fileSize, err := FileSize(filename)
	if Error(err) {
		return false, err
	}

	const max = 1000

	fileSize = Min(max, fileSize)

	ba := make([]byte, fileSize)

	f, err := os.Open(filename)
	if Error(err) {
		return false, err
	}
	defer func() {
		Error(f.Close())
	}()

	lr := io.LimitedReader{R: f, N: max}
	n, err := lr.Read(ba)
	if Error(err) {
		return false, err
	}

	ba = ba[:n]

	return IsTextContent(ba)
}
