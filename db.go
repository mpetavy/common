package common

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	dynamicstruct "github.com/ompluscator/dynamic-struct"
	"reflect"
	"strings"
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

type Rows struct {
	Columns []string
	Values  [][]string
	IsNull  [][]bool
	Fields  interface{}
}

func (database *Database) Query(sqlcmd string, args ...any) (*Rows, error) {
	ctx := context.Background()
	if database.Timeout != 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(context.Background(), database.Timeout)
		defer cancel()
	}

	query, err := database.currentDb().QueryContext(ctx, sqlcmd, args...)
	if Error(err) {
		return nil, err
	}
	defer func() {
		Error(query.Close())
	}()

	columns, err := query.Columns()
	if Error(err) {
		return nil, err
	}

	types, err := query.ColumnTypes()
	if Error(err) {
		return nil, err
	}

	raws := make([][]byte, len(columns))

	ptrRaws := make([]interface{}, len(columns))
	for i := range raws {
		ptrRaws[i] = &raws[i]
	}

	rows := &Rows{
		Columns: columns,
	}

	builder := dynamicstruct.NewStruct()

	for i := 0; i < len(columns); i++ {
		name := strings.ToUpper(columns[i])
		tag := fmt.Sprintf("`json:\"%s\"`", name)

		var v interface{}

		switch types[i].ScanType() {
		case reflect.TypeOf(sql.NullBool{}):
			v = true
		case reflect.TypeOf(sql.NullByte{}):
			v = byte(0)
		case reflect.TypeOf(sql.NullFloat64{}):
			v = float64(0)
		case reflect.TypeOf(sql.NullInt16{}):
			v = 0
		case reflect.TypeOf(sql.NullInt32{}):
			v = 0
		case reflect.TypeOf(sql.NullInt64{}):
			v = 0
		case reflect.TypeOf(sql.NullTime{}):
			v = time.Time{}
		default:
			v = ""
		}
		//switch types[i].ScanType() {
		//case reflect.TypeOf(sql.NullBool{}):
		//	v = sql.NullBool{}
		//case reflect.TypeOf(sql.NullByte{}):
		//	v = sql.NullByte{}
		//case reflect.TypeOf(sql.NullFloat64{}):
		//	v = sql.NullFloat64{}
		//case reflect.TypeOf(sql.NullInt16{}):
		//	v = sql.NullInt16{}
		//case reflect.TypeOf(sql.NullInt32{}):
		//	v = sql.NullInt32{}
		//case reflect.TypeOf(sql.NullInt64{}):
		//	v = sql.NullInt64{}
		//case reflect.TypeOf(sql.NullTime{}):
		//	v = sql.NullTime{}
		//default:
		//	v = sql.NullString{}
		//}

		builder.AddField(name, v, tag)
	}

	dynamicStruct := builder.Build()
	buf := bytes.Buffer{}

	for query.Next() {
		err = query.Scan(ptrRaws...)
		if Error(err) {
			return nil, err
		}

		values := make([]string, len(raws))

		for i, raw := range raws {
			if raw != nil {
				values[i] = string(raw)
			}
		}

		rows.Values = append(rows.Values, values)

		record := dynamicStruct.New()
		recordPtrs := make([]any, len(columns))
		recordElem := reflect.ValueOf(record).Elem()

		for i := 0; i < recordElem.NumField(); i++ {
			recordPtrs[i] = recordElem.Field(i).Addr().Interface()
		}

		err = query.Scan(recordPtrs...)
		if Error(err) {
			return nil, err
		}

		ba, err := json.MarshalIndent(record, "", "    ")
		if Error(err) {
			return nil, err
		}

		if buf.Len() == 0 {
			buf.WriteString("[")
		} else {
			buf.WriteString(",")
		}

		buf.Write(ba)
	}

	if buf.Len() > 0 {
		buf.WriteString("]")
	}

	rows.Fields = dynamicStruct.NewSliceOfStructs()

	err = json.Unmarshal(buf.Bytes(), &rows.Fields)
	if Error(err) {
		return nil, err
	}

	return rows, nil
}
