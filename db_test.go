//go:build unix

package common

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

func checkChanged(t *testing.T, db *Database, changed bool) {
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
	InitTesting(t)

	db, err := NewDatabase("sqlite3", "")
	assert.NoError(t, err)

	err = db.Open()
	assert.NoError(t, err)

	defer func() {
		assert.NoError(t, db.Close())
	}()

	stmts := []string{
		"create table foo (id integer not null primary key, name text)",
		"delete from foo",
	}

	for _, stmt := range stmts {
		_, err = db.Execute(stmt)
		assert.NoError(t, err)
	}

	for i := 0; i < 10000; i++ {
		_, err = db.Execute("insert into foo(id, name) values(?, ?)", i, fmt.Sprintf("こんにちは世界%03d", i))
		assert.NoError(t, err)
	}

	checkChanged(t, db, false)

	err = db.Begin()
	assert.NoError(t, err)

	_, err = db.Execute("update foo set name=?", "changed")
	assert.NoError(t, err)

	checkChanged(t, db, true)

	err = db.Rollback()
	assert.NoError(t, err)

	checkChanged(t, db, false)

	err = db.Begin()
	assert.NoError(t, err)

	_, err = db.Execute("update foo set name=?", "changed")
	assert.NoError(t, err)

	checkChanged(t, db, true)

	err = db.Commit()
	assert.NoError(t, err)

	checkChanged(t, db, true)
}

func TestScriptEngineDatabase(t *testing.T) {
	InitTesting(t)

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
	if Error(err) {
		return
	}

	err = engine.EnableDatabase()
	assert.Nil(t, err)

	_, err = engine.Run(time.Hour, "", "")

	assert.Nil(t, err)
}
