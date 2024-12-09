package sqldb

import (
	"fmt"
	//_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

const (
	ROW_COUNT  = 10000
	PAGE_COUNT = 123
)

func checkChanged(t *testing.T, db *SqlDb, changed bool) {
	resultset, err := db.Query("select id,name from foo order by id")
	require.NoError(t, err)

	require.Equal(t, []string{"ID", "NAME"}, resultset.ColumnNames)

	for i := 0; i < resultset.RowCount; i++ {
		id := resultset.FieldByName(i, "ID")
		name := resultset.FieldByName(i, "NAME")

		require.Equal(t, strconv.Itoa(i), id.String())
		if !changed {
			require.Equal(t, fmt.Sprintf("こんにちは世界%03d", i), name.String())
		} else {
			require.Equal(t, "changed", name.String())
		}
	}

	ids := make(map[string]struct{})
	for i := range ROW_COUNT {
		ids[strconv.Itoa(i)] = struct{}{}
	}

	fn := func(rs *Resultset) error {
		require.Equal(t, []string{"ID", "NAME"}, resultset.ColumnNames)
		require.LessOrEqual(t, rs.RowCount, PAGE_COUNT)

		for i := 0; i < rs.RowCount; i++ {
			id := rs.FieldByName(i, "ID")

			delete(ids, id.String())
		}

		return nil
	}

	rowCount, err := db.QueryPaged(fn, PAGE_COUNT, "select id,name from foo order by id")
	require.NoError(t, err)
	require.Equal(t, ROW_COUNT, rowCount)

	require.True(t, len(ids) == 0)

}

func TestSqlDb(t *testing.T) {
	database, err := NewSqlDb("sqlite3", "")
	require.NoError(t, err)

	err = database.Health()
	require.NoError(t, err)

	defer func() {
		require.NoError(t, database.Close())
	}()

	rs, err := database.Query("select sqlite_version()")
	require.NoError(t, err)

	version := rs.FieldByName(0, "sqlite_version")

	require.NotEqual(t, "", version)

	stmts := []string{
		"create table foo (id integer not null primary key, name text)",
		"delete from foo",
	}

	for _, stmt := range stmts {
		_, err = database.Execute(stmt)
		require.NoError(t, err)
	}

	for i := 0; i < ROW_COUNT; i++ {
		_, err = database.Execute("insert into foo(id, name) values(?, ?)", i, fmt.Sprintf("こんにちは世界%03d", i))
		require.NoError(t, err)
	}

	checkChanged(t, database, false)

	err = database.Begin()
	require.NoError(t, err)

	_, err = database.Execute("update foo set name=?", "changed")
	require.NoError(t, err)

	checkChanged(t, database, true)

	err = database.Rollback()
	require.NoError(t, err)

	checkChanged(t, database, false)

	err = database.Begin()
	require.NoError(t, err)

	_, err = database.Execute("update foo set name=?", "changed")
	require.NoError(t, err)

	checkChanged(t, database, true)

	err = database.Commit()
	require.NoError(t, err)

	checkChanged(t, database, true)
}

//func testConn(t *testing.T, msg string, database *SqlDb) error {
//	tty, err := tty.Open()
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer tty.Close()
//
//	fmt.Printf("%s\n", msg)
//
//	_, err = tty.ReadRune()
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	rs, err := database.Query("select version()")
//	if err != nil {
//		return err
//	}
//
//	fmt.Printf("%s\n", rs.FieldByIndex(0, 0).String())
//
//	return nil
//}
//
//func TestSqlDbReconnect(t *testing.T) {
//	t.Skipf("must be run on console")
//
//	dsn := "host=localhost port=5432 user=postgres password=postgres dbname=postgres"
//
//	database, err := NewSqlDb("pgx", dsn)
//	require.NoError(t, err)
//
//	err = database.Health()
//	require.NoError(t, err)
//
//	defer func() {
//		require.NoError(t, database.Close())
//	}()
//
//	require.NoError(t, testConn(t, "start db", database))
//
//	require.Error(t, testConn(t, "now stop db", database))
//
//	require.NoError(t, testConn(t, "and now start db again", database))
//}
