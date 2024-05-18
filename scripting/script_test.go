package scripting

import (
	"fmt"
	"github.com/mpetavy/common"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScriptEngine(t *testing.T) {
	common.InitTesting(t)

	files := []string{"a.js", "b.js"}

	for _, file := range files {
		modulePath := filepath.Join(os.TempDir(), file)
		src := fmt.Sprintf(`
function test() {
	return '%s'; 
}
exports.test = test;
`, file)

		err := os.WriteFile(modulePath, []byte(src), common.DefaultFileMode)
		if common.Error(err) {
			return
		}
	}

	src := `
var a = require('a.js');
var b = require('b.js');
a.test() + ';' + b.test();
`

	engine, err := NewScriptEngine(src, os.TempDir())
	if common.Error(err) {
		return
	}

	v, err := engine.Run(time.Second*3, "", nil)
	if common.Error(err) {
		return
	}

	assert.Nil(t, err)
	assert.True(t, strings.Contains(v.String(), "a.js"))
	assert.True(t, strings.Contains(v.String(), "b.js"))
}

func TestScriptEngineTimeout(t *testing.T) {
	common.InitTesting(t)

	src := "while(true) {}"

	engine, err := NewScriptEngine(src, "")
	if common.Error(err) {
		return
	}

	_, err = engine.Run(time.Second, "", nil)

	assert.NotNil(t, err)
	assert.True(t, common.IsErrTimeout(err))
}

func TestScriptEngineException(t *testing.T) {
	common.InitTesting(t)

	msg := "EXCEPTION!"

	src := fmt.Sprintf("throw new Error('%s');", msg)

	engine, err := NewScriptEngine(src, "")
	if common.Error(err) {
		return
	}

	_, err = engine.Run(time.Second, "", nil)

	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), msg))
}

func TestScriptEngineArgs(t *testing.T) {
	common.InitTesting(t)

	src := `
function main(args) {
let input = args.input;
args.output = "hello " + input;
}
`

	engine, err := NewScriptEngine(src, "")
	if common.Error(err) {
		return
	}

	args := make(map[string]any)
	args["input"] = "world"

	_, err = engine.Run(time.Second, "main", args)

	assert.Nil(t, err)
	assert.NotNil(t, args["output"])
	assert.Equal(t, "hello world", args["output"])
}

func TestScriptEngineFormatJavascript(t *testing.T) {
	common.InitTesting(t)

	src := `
function main(args) {
let input = args.input;
args.output = "hello " + input;
}
`

	_, err := FormatJavascriptCode(src)
	if common.Error(err) {
		return
	}

	assert.Nil(t, err)
}

func TestScriptEngineEtree(t *testing.T) {
	common.InitTesting(t)

	src := `
d = Object.create(etree);
r = d.CreateElement('root');
r.CreateAttr('name','foo');

console.log(d.WriteToString());
`
	engine, err := NewScriptEngine(src, "")
	if common.Error(err) {
		return
	}

	_, err = engine.Run(time.Hour, "", nil)

	assert.Nil(t, err)
}

func TestScriptEngineHL7(t *testing.T) {
	common.InitTesting(t)

	tests := []struct {
		file     string
		expected string
	}{
		{
			file:     "./testdata/node/test-hl7-standard.js",
			expected: fmt.Sprintf("MSH|^~\\&|Example|123456|||%s||ADT^A08||T|2.3|", time.Now().Format(common.Year+common.Month+common.Day)),
		},
		{
			file: "./testdata/node/test-hl7-standard-2.js",
			expected: fmt.Sprintf("%s\r\n%s\r\n%s\r\n%s", "MSH|^~\\&|EPIC|EPICADT|SMS|SMSADT|199912271408|CHARRIS|ADT^A04|1817457|D|2.5||",
				"PID||0493575^^^2^ID 1|454721||DOE^JOHN^^^^|DOE^JOHN^^^^|19480203|M||B|254 MYSTREET AVE^^MYTOWN^OH^44123^USA||(216)123-4567|||M|NON|400003403~1129086||",
				"NK1||ROE^MARIE^^^^|SPO||(216)123-4567||EC||||||||||||||||||||||||||||",
				"PV1||O|168 ~219~C~PMA^^^^^^^^^||||277^ALLEN MYLASTNAME^BONNIE^^^^|||||||||| ||2688684|||||||||||||||||||||||||199912271408||||||002376853|"),
		},
		{
			file:     "./testdata/node/test-json-stringify.js",
			expected: "{\"Interests\":[\"football\",\"hiking\",\"gym\"],\"Address\":{\"Name\":\"ransom\",\"Street\":\"Mystreet 17\",\"City\":\"Mytown\",\"Birthday\":\"Fri Apr 05 2024 13:45:14 GMT+0200 (CEST)\"}}",
		},
		{
			file:     "./testdata/node/test-xml.js",
			expected: "",
		},
	}

	for _, test := range tests {
		if !t.Run(test.file, func(t *testing.T) {
			common.InitTesting(t)

			src, err := os.ReadFile(test.file)
			if common.Error(err) {
				return
			}

			se, err := NewScriptEngine(string(src), "./testdata/node/node_modules")
			if common.Error(err) {
				return
			}

			output, err := se.Run(time.Second*3, "", "")
			if common.Error(err) {
				return
			}

			if test.expected != "" {
				assert.Equal(t, test.expected, output.String())
			}
		}) {
			return
		}
	}
}
