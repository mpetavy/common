package common

import (
	"context"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

type Database struct {
	Driver      string
	DSN         string
	Timeout     time.Duration
	Isolation   sql.IsolationLevel
	db          *sql.DB
	txCounter   int
	tx          *sql.Tx
	txCtx       context.Context
	txCtxCancel context.CancelFunc
}

type dbintf interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func DatabaseDrivers() []string {
	return sql.Drivers()
}

func NewDatabase(driver, dsn string) (*Database, error) {
	db := &Database{}

	err := db.Init(driver, dsn)
	if Error(err) {
		return nil, err
	}

	return db, nil
}

func (database *Database) currentDb() dbintf {
	if database.tx != nil {
		return database.tx
	} else {
		return database.db
	}
}

func (database *Database) Init(driver, dsn string) error {
	database.Driver = driver
	database.DSN = dsn
	database.Timeout = time.Minute
	database.Isolation = sql.LevelReadCommitted

	return nil
}

func (database *Database) Open() error {
	db, err := sql.Open(database.Driver, database.DSN)
	if Error(err) {
		return err
	}

	ctx := context.Background()
	if database.Timeout != 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(context.Background(), database.Timeout)
		defer cancel()
	}

	err = db.PingContext(ctx)
	if Error(err) {
		return err
	}

	database.db = db

	return nil
}

func (database *Database) Close() error {
	err := database.db.Close()
	if Error(err) {
		return err
	}

	database.db = nil

	return nil
}

func (database *Database) Begin() error {
	database.txCounter++

	if database.txCounter > 1 {
		return nil
	}

	database.txCtx, database.txCtxCancel = context.WithCancel(context.Background())

	var err error

	database.tx, err = database.db.BeginTx(database.txCtx, &sql.TxOptions{Isolation: database.Isolation})
	if Error(err) {
		return err
	}

	return nil
}

func (database *Database) Rollback() error {
	database.txCounter--

	if database.txCounter > 0 {
		return nil
	}

	err := database.tx.Rollback()
	if Error(err) {
		return err
	}

	database.txCtxCancel()

	database.tx = nil
	database.txCtx = nil
	database.txCtxCancel = nil

	return nil
}

func (database *Database) Commit() error {
	database.txCounter--

	if database.txCounter > 0 {
		return nil
	}

	err := database.tx.Commit()
	if Error(err) {
		return err
	}

	database.txCtxCancel()

	database.tx = nil
	database.txCtx = nil
	database.txCtxCancel = nil

	return nil
}

func (database *Database) Execute(sqlcmd string, args ...any) (int64, error) {
	ctx := context.Background()
	if database.Timeout != 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(context.Background(), database.Timeout)
		defer cancel()
	}

	result, err := database.currentDb().ExecContext(ctx, sqlcmd, args...)
	if Error(err) {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if Error(err) {
		return 0, err
	}

	return rowsAffected, nil
}

func (database *Database) Query(sqlcmd string, args ...any) ([]string, [][]string, error) {
	ctx := context.Background()
	if database.Timeout != 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(context.Background(), database.Timeout)
		defer cancel()
	}

	query, err := database.currentDb().QueryContext(ctx, sqlcmd, args...)
	if Error(err) {
		return nil, nil, err
	}
	defer func() {
		Error(query.Close())
	}()

	columns, err := query.Columns()
	if Error(err) {
		return nil, nil, err
	}

	rawResult := make([][]byte, len(columns))

	dest := make([]interface{}, len(columns))
	for i := range rawResult {
		dest[i] = &rawResult[i]
	}

	rows := make([][]string, 0)

	for query.Next() {
		err = query.Scan(dest...)
		if Error(err) {
			return nil, nil, err
		}

		strResult := make([]string, len(columns))

		for i, raw := range rawResult {
			if raw == nil {
				strResult[i] = "<null>"
			} else {
				strResult[i] = string(raw)
			}
		}

		rows = append(rows, strResult)
	}

	return columns, rows, nil
}
