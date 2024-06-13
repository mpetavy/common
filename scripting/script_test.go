package scripting

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mpetavy/common"
	"github.com/mpetavy/common/sqldb"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"strconv"
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

func checkChanged(t *testing.T, db *sqldb.Database, changed bool) {
	resultset, err := db.Query("select id,name from foo order by id")
	assert.NoError(t, err)

	assert.Equal(t, []string{"ID", "NAME"}, resultset.ColumnNames)

	for i := 0; i < resultset.RowCount; i++ {
		id, err := resultset.Get(i, "ID")
		assert.NoError(t, err)
		name, err := resultset.Get(i, "NAME")
		assert.NoError(t, err)

		assert.Equal(t, strconv.Itoa(i), id.String())
		if !changed {
			assert.Equal(t, fmt.Sprintf("こんにちは世界%03d", i), name.String())
		} else {
			assert.Equal(t, "changed", name.String())
		}
	}
}

func TestDb(t *testing.T) {
	common.InitTesting(t)

	database, err := sqldb.NewDatabase("sqlite3", "")
	assert.NoError(t, err)

	err = database.Open()
	assert.NoError(t, err)

	defer func() {
		assert.NoError(t, database.Close())
	}()

	rs, err := database.Query("select sqlite_version()")
	assert.NoError(t, err)

	version, err := rs.Get(0, "sqlite_version")
	assert.NoError(t, err)

	assert.NotEqual(t, "", version)

	stmts := []string{
		"create table foo (id integer not null primary key, name text)",
		"delete from foo",
	}

	for _, stmt := range stmts {
		_, err = database.Execute(stmt)
		assert.NoError(t, err)
	}

	for i := 0; i < 10000; i++ {
		_, err = database.Execute("insert into foo(id, name) values(?, ?)", i, fmt.Sprintf("こんにちは世界%03d", i))
		assert.NoError(t, err)
	}

	checkChanged(t, database, false)

	err = database.Begin()
	assert.NoError(t, err)

	_, err = database.Execute("update foo set name=?", "changed")
	assert.NoError(t, err)

	checkChanged(t, database, true)

	err = database.Rollback()
	assert.NoError(t, err)

	checkChanged(t, database, false)

	err = database.Begin()
	assert.NoError(t, err)

	_, err = database.Execute("update foo set name=?", "changed")
	assert.NoError(t, err)

	checkChanged(t, database, true)

	err = database.Commit()
	assert.NoError(t, err)

	checkChanged(t, database, true)
}

func TestScriptEngineDatabase(t *testing.T) {
	common.InitTesting(t)

	src := `
var db = database;
db.init('sqlite3','');
db.open();
db.execute('create table foo (id integer not null primary key, name text,empty text)');
db.execute('insert into foo (id, name, empty) values (?, ?, ?)',123,'test123','abc');
db.execute('insert into foo (id, name, empty) values (?, ?, ?)',456,'test456',null);
db.execute('insert into foo (id, name, empty) values (?, ?, ?)',789,'test789','cde');
var result = db.query('select * from foo');
console.log('The query returns ' + result.Rows.length + ' rows');
console.log('The query returns the following colums: ' + result.ColumnNames);
for(var i = 0;i < result.Rows.length;i++) {
  console.log('------- Row #' + i + '-----');
  console.log(result.Rows[i].ID.Value);
  console.log(result.Rows[i].NAME.Value);
  console.log(result.Rows[i].EMPTY.Value);
  console.log(result.Rows[i].EMPTY.IsNull);
}
db.close();
`

	engine, err := NewScriptEngine(src, "")
	if common.Error(err) {
		return
	}

	err = EnableDatabase(engine)
	assert.Nil(t, err)

	_, err = engine.Run(time.Hour, "", "")

	assert.Nil(t, err)
}
