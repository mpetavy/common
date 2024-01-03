package db

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mpetavy/common"
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
	if common.Error(err) {
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
	if common.Error(err) {
		return err
	}

	ctx := context.Background()
	if database.Timeout != 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(context.Background(), database.Timeout)
		defer cancel()
	}

	err = db.PingContext(ctx)
	if common.Error(err) {
		return err
	}

	database.db = db

	return nil
}

func (database *Database) Close() error {
	err := database.db.Close()
	if common.Error(err) {
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
	if common.Error(err) {
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
	if common.Error(err) {
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
	if common.Error(err) {
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
	if common.Error(err) {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if common.Error(err) {
		return 0, err
	}

	return rowsAffected, nil
}

type Field struct {
	Value  any  `json:"value"`
	IsNull bool `json:"isnull"`
}

func (f *Field) String() string {
	if f.IsNull {
		return ""
	}

	return fmt.Sprintf("%v", f.Value)
}

type Resultset struct {
	ColumnNames []string
	RowCount    int
	Rows        any
}

func (rs *Resultset) Get(row int, fieldName string) (Field, error) {
	resultsetValue := reflect.ValueOf(rs.Rows)
	rowValue := resultsetValue.Elem().Index(row)
	colValue := rowValue.FieldByName(fieldName)

	return colValue.Interface().(Field), nil
}

func (database *Database) Query(sqlcmd string, args ...any) (*Resultset, error) {
	ctx := context.Background()
	if database.Timeout != 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(context.Background(), database.Timeout)
		defer cancel()
	}

	query, err := database.currentDb().QueryContext(ctx, sqlcmd, args...)
	if common.Error(err) {
		return nil, err
	}
	defer func() {
		common.Error(query.Close())
	}()

	columnNames, err := query.Columns()
	if common.Error(err) {
		return nil, err
	}
	for i := 0; i < len(columnNames); i++ {
		columnNames[i] = strings.ToUpper(columnNames[i])
	}

	columnTypes, err := query.ColumnTypes()
	if common.Error(err) {
		return nil, err
	}

	rows := &Resultset{
		ColumnNames: columnNames,
	}

	scanBuilder := dynamicstruct.NewStruct()
	recordBuilder := dynamicstruct.NewStruct()

	for i := 0; i < len(columnNames); i++ {
		name := columnNames[i]
		tag := fmt.Sprintf("`json:\"%s\"`", name)

		var scanField interface{}
		var recordField interface{}

		switch columnTypes[i].ScanType() {
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

		field := Field{
			Value:  recordField,
			IsNull: false,
		}

		scanBuilder.AddField(name, scanField, tag)
		recordBuilder.AddField(name, field, tag)
	}

	scanStruct := scanBuilder.Build()
	recordStruct := recordBuilder.Build()

	recordBuf := bytes.Buffer{}

	for query.Next() {
		scan := scanStruct.New()
		scanPtrs := make([]any, len(columnNames))
		scanElem := reflect.ValueOf(scan).Elem()

		for i := 0; i < scanElem.NumField(); i++ {
			scanPtrs[i] = scanElem.Field(i).Addr().Interface()
		}

		err = query.Scan(scanPtrs...)
		if common.Error(err) {
			return nil, err
		}

		record := recordStruct.New()
		recordElem := reflect.ValueOf(record).Elem()

		for i := 0; i < scanElem.NumField(); i++ {
			recordField := recordElem.FieldByName(columnNames[i])
			recordFieldIsNull := recordField.FieldByName("IsNull")
			recordFieldValue := recordField.FieldByName("Value")

			switch columnTypes[i].ScanType() {
			case reflect.TypeOf(sql.NullBool{}):
				s, ok := scanPtrs[i].(*sql.NullBool)
				isNull := !(ok && s.Valid)
				recordFieldIsNull.Set(reflect.ValueOf(isNull))
				recordFieldIsNull.Set(reflect.ValueOf(!(ok && s.Valid)))
				if ok && s.Valid {
					recordFieldValue.Set(reflect.ValueOf(s.Bool))
				}
			case reflect.TypeOf(sql.NullByte{}):
				s, ok := scanPtrs[i].(*sql.NullByte)
				isNull := !(ok && s.Valid)
				recordFieldIsNull.Set(reflect.ValueOf(isNull))
				recordFieldIsNull.Set(reflect.ValueOf(!(ok && s.Valid)))
				if ok && s.Valid {
					recordFieldValue.Set(reflect.ValueOf(s.Byte))
				}
			case reflect.TypeOf(sql.NullFloat64{}):
				s, ok := scanPtrs[i].(*sql.NullFloat64)
				isNull := !(ok && s.Valid)
				recordFieldIsNull.Set(reflect.ValueOf(isNull))
				recordFieldIsNull.Set(reflect.ValueOf(!(ok && s.Valid)))
				if ok && s.Valid {
					recordFieldValue.Set(reflect.ValueOf(s.Float64))
				}
			case reflect.TypeOf(sql.NullInt16{}):
				s, ok := scanPtrs[i].(*sql.NullInt16)
				isNull := !(ok && s.Valid)
				recordFieldIsNull.Set(reflect.ValueOf(isNull))
				recordFieldIsNull.Set(reflect.ValueOf(!(ok && s.Valid)))
				if ok && s.Valid {
					recordFieldValue.Set(reflect.ValueOf(s.Int16))
				}
			case reflect.TypeOf(sql.NullInt32{}):
				s, ok := scanPtrs[i].(*sql.NullInt32)
				isNull := !(ok && s.Valid)
				recordFieldIsNull.Set(reflect.ValueOf(isNull))
				recordFieldIsNull.Set(reflect.ValueOf(!(ok && s.Valid)))
				if ok && s.Valid {
					recordFieldValue.Set(reflect.ValueOf(s.Int32))
				}
			case reflect.TypeOf(sql.NullInt64{}):
				s, ok := scanPtrs[i].(*sql.NullInt64)
				isNull := !(ok && s.Valid)
				recordFieldIsNull.Set(reflect.ValueOf(isNull))
				recordFieldIsNull.Set(reflect.ValueOf(!(ok && s.Valid)))
				if ok && s.Valid {
					recordFieldValue.Set(reflect.ValueOf(s.Int64))
				}
			case reflect.TypeOf(sql.NullTime{}):
				s, ok := scanPtrs[i].(*sql.NullTime)
				isNull := !(ok && s.Valid)
				recordFieldIsNull.Set(reflect.ValueOf(isNull))
				recordFieldIsNull.Set(reflect.ValueOf(!(ok && s.Valid)))
				if ok && s.Valid {
					recordFieldValue.Set(reflect.ValueOf(s.Time))
				}
			case reflect.TypeOf(sql.NullString{}):
				s, ok := scanPtrs[i].(*sql.NullString)
				isNull := !(ok && s.Valid)
				recordFieldIsNull.Set(reflect.ValueOf(isNull))
				recordFieldIsNull.Set(reflect.ValueOf(!(ok && s.Valid)))
				if ok && s.Valid {
					recordFieldValue.Set(reflect.ValueOf(s.String))
				}
			}
		}

		ba, err := json.MarshalIndent(record, "", "    ")
		if common.Error(err) {
			return nil, err
		}

		if recordBuf.Len() == 0 {
			recordBuf.WriteString("[")
		} else {
			recordBuf.WriteString(",")
		}

		recordBuf.Write(ba)
	}

	if recordBuf.Len() > 0 {
		recordBuf.WriteString("]")
	}

	rows.Rows = recordStruct.NewSliceOfStructs()

	err = json.Unmarshal(recordBuf.Bytes(), &rows.Rows)
	if common.Error(err) {
		return nil, err
	}

	return rows, nil
}
