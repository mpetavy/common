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
		src := `
function test() {
	return 'filename:' + __filename + ';dirname:' + __dirname; 
}
exports.test = test;
`

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

	v, err := engine.Run(time.Second, "", "")
	if Error(err) {
		return
	}

	assert.Nil(t, err)
	assert.True(t, strings.Contains(v, "a.js"))
	assert.True(t, strings.Contains(v, "b.js"))
}

func TestScriptEngineTimeout(t *testing.T) {
	InitTesting(t)

	src := "while(true) {}"

	engine, err := NewScriptEngine(src, "")
	if Error(err) {
		return
	}

	_, err = engine.Run(time.Second, "", "")

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

	_, err = engine.Run(time.Second, "", "")

	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), msg))
}

func TestScriptEngineDatabase(t *testing.T) {
	InitTesting(t)

	src := `
var db = database;
db.init('sqlite3','');
db.open();
db.execute('create table foo (id integer not null primary key, name text)');
db.execute('insert into foo(id, name) values(?, ?)',123,'test');
var result = db.query('select * from foo');
// result is a JS object with 2 properties. You can acces columns by [0] and records by [1] 
console.log("-- Columns -------------");
console.table(result[0]);
console.log("-- Records -------------");
console.table(result[1]);
db.close();
`

	engine, err := NewScriptEngine(src, "")
	if Error(err) {
		return
	}

	err = engine.EnableDatabase()
	assert.Nil(t, err)

	_, err = engine.Run(time.Second, "", "")

	assert.Nil(t, err)
}
