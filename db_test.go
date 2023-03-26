package common

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func checkChanged(t *testing.T, db *Database, changed bool) {
	rows, err := db.Query("select id,name from foo order by id")
	assert.NoError(t, err)

	assert.Equal(t, []string{"id", "name"}, rows.Columns)

	for i := 0; i < 3; i++ {
		assert.Equal(t, strconv.Itoa(i), rows.Values[i][0])
		if !changed {
			assert.Equal(t, fmt.Sprintf("こんにちは世界%03d", i), rows.Values[i][1])
		} else {
			assert.Equal(t, "changed", rows.Values[i][1])
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

	for i := 0; i < 3; i++ {
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
