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
	IsNull  any
	Fields  any
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

	scanBuilder := dynamicstruct.NewStruct()
	recordBuilder := dynamicstruct.NewStruct()
	nullBuilder := dynamicstruct.NewStruct()

	for i := 0; i < len(columns); i++ {
		name := strings.ToUpper(columns[i])
		tag := fmt.Sprintf("`json:\"%s\"`", name)

		var scanField interface{}
		var recordField interface{}

		switch types[i].ScanType() {
		case reflect.TypeOf(sql.NullBool{}):
			scanField = sql.NullBool{}
			recordField = true
		case reflect.TypeOf(sql.NullByte{}):
			scanField = sql.NullByte{}
			recordField = byte(0)
		case reflect.TypeOf(sql.NullFloat64{}):
			scanField = sql.NullFloat64{}
			recordField = float64(0)
		case reflect.TypeOf(sql.NullInt16{}):
			scanField = sql.NullInt16{}
			recordField = 0
		case reflect.TypeOf(sql.NullInt32{}):
			scanField = sql.NullInt32{}
			recordField = 0
		case reflect.TypeOf(sql.NullInt64{}):
			scanField = sql.NullInt64{}
			recordField = 0
		case reflect.TypeOf(sql.NullTime{}):
			scanField = sql.NullTime{}
			recordField = time.Time{}
		default:
			scanField = sql.NullString{}
			recordField = ""
		}

		scanBuilder.AddField(name, scanField, tag)
		recordBuilder.AddField(name, recordField, tag)
		nullBuilder.AddField(name, false, tag)
	}

	scanStruct := scanBuilder.Build()
	recordStruct := recordBuilder.Build()
	nullStruct := nullBuilder.Build()

	recordBuf := bytes.Buffer{}
	nullBuf := bytes.Buffer{}

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

		scan := scanStruct.New()
		scanPtrs := make([]any, len(columns))
		scanElem := reflect.ValueOf(scan).Elem()

		for i := 0; i < scanElem.NumField(); i++ {
			scanPtrs[i] = scanElem.Field(i).Addr().Interface()
		}

		err = query.Scan(scanPtrs...)
		if Error(err) {
			return nil, err
		}

		record := recordStruct.New()
		recordElem := reflect.ValueOf(record).Elem()
		null := nullStruct.New()
		nullElem := reflect.ValueOf(null).Elem()

		for i := 0; i < scanElem.NumField(); i++ {
			switch types[i].ScanType() {
			case reflect.TypeOf(sql.NullBool{}):
				s, ok := scanPtrs[i].(*sql.NullBool)
				if ok && s.Valid {
					recordElem.Field(i).Set(reflect.ValueOf(s.Bool))
				}
				nullElem.Field(i).Set(reflect.ValueOf(!(ok && s.Valid)))
			case reflect.TypeOf(sql.NullByte{}):
				s, ok := scanPtrs[i].(*sql.NullByte)
				if ok && s.Valid {
					recordElem.Field(i).Set(reflect.ValueOf(int(s.Byte)))
				}
				nullElem.Field(i).Set(reflect.ValueOf(!(ok && s.Valid)))
			case reflect.TypeOf(sql.NullFloat64{}):
				s, ok := scanPtrs[i].(*sql.NullFloat64)
				if ok && s.Valid {
					recordElem.Field(i).Set(reflect.ValueOf(float64(s.Float64)))
				}
				nullElem.Field(i).Set(reflect.ValueOf(!(ok && s.Valid)))
			case reflect.TypeOf(sql.NullInt16{}):
				s, ok := scanPtrs[i].(*sql.NullInt16)
				if ok && s.Valid {
					recordElem.Field(i).Set(reflect.ValueOf(int(s.Int16)))
				}
				nullElem.Field(i).Set(reflect.ValueOf(!(ok && s.Valid)))
			case reflect.TypeOf(sql.NullInt32{}):
				s, ok := scanPtrs[i].(*sql.NullInt32)
				if ok && s.Valid {
					recordElem.Field(i).Set(reflect.ValueOf(int(s.Int32)))
				}
				nullElem.Field(i).Set(reflect.ValueOf(!(ok && s.Valid)))
			case reflect.TypeOf(sql.NullInt64{}):
				s, ok := scanPtrs[i].(*sql.NullInt64)
				if ok && s.Valid {
					recordElem.Field(i).Set(reflect.ValueOf(int(s.Int64)))
				}
				nullElem.Field(i).Set(reflect.ValueOf(!(ok && s.Valid)))
			case reflect.TypeOf(sql.NullTime{}):
				s, ok := scanPtrs[i].(*sql.NullTime)
				if ok && s.Valid {
					recordElem.Field(i).Set(reflect.ValueOf(s.Time))
				}
				nullElem.Field(i).Set(reflect.ValueOf(!(ok && s.Valid)))
			case reflect.TypeOf(sql.NullString{}):
				s, ok := scanPtrs[i].(*sql.NullString)
				if ok && s.Valid {
					recordElem.Field(i).Set(reflect.ValueOf(s.String))
				}
				nullElem.Field(i).Set(reflect.ValueOf(!(ok && s.Valid)))
			}
		}

		ba, err := json.MarshalIndent(record, "", "    ")
		if Error(err) {
			return nil, err
		}

		if recordBuf.Len() == 0 {
			recordBuf.WriteString("[")
		} else {
			recordBuf.WriteString(",")
		}

		recordBuf.Write(ba)

		ba, err = json.MarshalIndent(null, "", "    ")
		if Error(err) {
			return nil, err
		}

		if nullBuf.Len() == 0 {
			nullBuf.WriteString("[")
		} else {
			nullBuf.WriteString(",")
		}

		nullBuf.Write(ba)
	}

	if recordBuf.Len() > 0 {
		recordBuf.WriteString("]")
	}

	if nullBuf.Len() > 0 {
		nullBuf.WriteString("]")
	}

	rows.Fields = recordStruct.NewSliceOfStructs()
	rows.IsNull = nullStruct.NewSliceOfStructs()

	err = json.Unmarshal(recordBuf.Bytes(), &rows.Fields)
	if Error(err) {
		return nil, err
	}

	err = json.Unmarshal(nullBuf.Bytes(), &rows.IsNull)
	if Error(err) {
		return nil, err
	}

	return rows, nil
}
