package common

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScriptEngine(t *testing.T) {
	InitTesting(t)

	files := []string{"a.js", "b.js"}

	for _, file := range files {
		modulePath := filepath.Join(os.TempDir(), file)
		src := fmt.Sprintf(`
function test() {
	return '%s'; 
}
exports.test = test;
`, file)

		err := os.WriteFile(modulePath, []byte(src), DefaultFileMode)
		if Error(err) {
			return
		}
	}

	src := `
var a = require('a.js');
var b = require('b.js');
a.test() + ';' + b.test();
`

	engine, err := NewScriptEngine(src, os.TempDir())
	if Error(err) {
		return
	}

	v, err := engine.Run(time.Second, "", nil)
	if Error(err) {
		return
	}

	assert.Nil(t, err)
	assert.True(t, strings.Contains(v.String(), "a.js"))
	assert.True(t, strings.Contains(v.String(), "b.js"))
}

func TestScriptEngineTimeout(t *testing.T) {
	InitTesting(t)

	src := "while(true) {}"

	engine, err := NewScriptEngine(src, "")
	if Error(err) {
		return
	}

	_, err = engine.Run(time.Second, "", nil)

	assert.NotNil(t, err)
	assert.True(t, IsErrTimeout(err))
}

func TestScriptEngineException(t *testing.T) {
	InitTesting(t)

	msg := "EXCEPTION!"

	src := fmt.Sprintf("throw new Error('%s');", msg)

	engine, err := NewScriptEngine(src, "")
	if Error(err) {
		return
	}

	_, err = engine.Run(time.Second, "", nil)

	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), msg))
}

func TestScriptEngineArgs(t *testing.T) {
	InitTesting(t)

	src := `
function main(args) {
let input = args.input;
args.output = "hello " + input;
}
`

	engine, err := NewScriptEngine(src, "")
	if Error(err) {
		return
	}

	args := make(map[string]any)
	args["input"] = "world"

	_, err = engine.Run(time.Second, "main", args)

	assert.Nil(t, err)
	assert.NotNil(t, args["output"])
	assert.Equal(t, "hello world", args["output"])
}

func TestScriptFormatJavascript(t *testing.T) {
	InitTesting(t)

	src := `
function main(args) {
let input = args.input;
args.output = "hello " + input;
}
`

	_, err := FormatJavascriptCode(src)
	if Error(err) {
		return
	}

	assert.Nil(t, err)
}

func TestHttp(t *testing.T) {
	InitTesting(t)

	src := `
d = Object.create(etree);
r = d.CreateElement('root');
r.CreateAttr('name','Marcel');

console.print('%s\n',d.WriteToString());
`
	//	src := `
	//m = new Map();
	//m.set('Content-Type',['application/xml']);
	//
	//let get = http.execute('GET','https://192.168.1.35:8090/api/v1/czmxml?locale=en&type=KER','hadern','hadern',null,null);
	//let getbody = http.body(get);
	//console.printf(String.fromCharCode(...getbody));
	//let post = http.execute('POST','https://192.168.1.35:8090/api/v1/czmxml?locale=en&type=KER','hadern','hadern',m,getbody);
	//let postbody = http.body(post);
	//console.printf(String.fromCharCode(...postbody));
	//`
	//src := fmt.Sprintf("c2 = Object.create(console);c2.printf('Hello world!');")
	//src := fmt.Sprintf("msg = http.execute('https://www.google.de');console.printf('+++\\n%%s\\n---\\n',msg);")
	//src := fmt.Sprintf("console.info('https://www.google.de');")

	engine, err := NewScriptEngine(src, "")
	if Error(err) {
		return
	}

	_, err = engine.Run(time.Hour, "", nil)

	assert.Nil(t, err)
}
