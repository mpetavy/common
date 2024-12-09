package sqldb

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/mpetavy/common"
	dynamicstruct "github.com/ompluscator/dynamic-struct"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type SqlDb struct {
	Driver      string
	DSN         string
	conn        *sql.DB
	txCounter   int
	tx          *sql.Tx
	txCtx       context.Context
	txCtxCancel context.CancelFunc
}

const (
	FlagNameDbPingTimeout  = "db.ping.timeout"
	FlagNameDbQueryTimeout = "db.query.timeout"
	FlagNameDbMaxIdle      = "db.max.idle"
	FlagNameDbMaxOpen      = "db.max.open"
	FlagNameDbMaxLifetime  = "db.max.lifetime"

	isolation = sql.LevelReadCommitted
)

var (
	FlagDbPingTimeout  = common.SystemFlagInt(FlagNameDbPingTimeout, 3*1000, "Database ping timeout")
	FlagDbQueryTimeout = common.SystemFlagInt(FlagNameDbQueryTimeout, 120*1000, "Database query timeout")
	FlagDbMaxIdle      = common.SystemFlagInt(FlagNameDbMaxIdle, 0, "Database max idle connections")
	FlagMaxOpen        = common.SystemFlagInt(FlagNameDbMaxOpen, 0, "Database max open connections")
	FlagMaxLifetime    = common.SystemFlagInt(FlagNameDbMaxLifetime, 0, "Database connection max lifetime")

	regexFieldName = regexp.MustCompile("([\\w\\d_]+)")
)

type dbintf interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func NewSqlDb(driver, dsn string) (*SqlDb, error) {
	sqlDb := &SqlDb{}

	err := sqlDb.Init(driver, dsn)
	if common.Error(err) {
		return nil, err
	}

	return sqlDb, nil
}

func (sqlDb *SqlDb) Init(driver, dsn string) error {
	sqlDb.Driver = driver
	sqlDb.DSN = dsn

	return nil
}

func (sqlDb *SqlDb) currentDb() dbintf {
	if sqlDb.tx != nil {
		return sqlDb.tx
	} else {
		return sqlDb.conn
	}
}

func (sqlDb *SqlDb) open() error {
	if sqlDb.conn == nil {
		var err error

		sqlDb.conn, err = sql.Open(sqlDb.Driver, sqlDb.DSN)
		if common.Error(err) {
			return err
		}

		if *FlagDbMaxIdle > 0 {
			sqlDb.conn.SetMaxIdleConns(*FlagDbMaxIdle)
		}
		if *FlagMaxLifetime > 0 {
			sqlDb.conn.SetConnMaxLifetime(common.MillisecondToDuration(*FlagMaxLifetime))
		}
		if *FlagMaxOpen > 0 {
			sqlDb.conn.SetMaxOpenConns(*FlagMaxOpen)
		}
	}

	ctx := context.Background()
	if *FlagDbPingTimeout != 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(context.Background(), common.MillisecondToDuration(*FlagDbPingTimeout))
		defer cancel()
	}

	err := sqlDb.conn.PingContext(ctx)
	if common.Error(err) {
		return err
	}

	return nil
}

func (sqlDb *SqlDb) Close() error {
	if sqlDb.conn == nil {
		return nil
	}

	err := sqlDb.conn.Close()
	if common.Error(err) {
		return err
	}

	sqlDb.conn = nil

	return nil
}

func (sqlDb *SqlDb) Health() error {
	err := sqlDb.open()
	if common.Error(err) {
		return err
	}

	return nil
}

func (sqlDb *SqlDb) Begin() error {
	err := sqlDb.open()
	if common.Error(err) {
		return err
	}

	sqlDb.txCounter++

	if sqlDb.txCounter > 1 {
		return nil
	}

	sqlDb.txCtx, sqlDb.txCtxCancel = context.WithCancel(context.Background())

	sqlDb.tx, err = sqlDb.conn.BeginTx(sqlDb.txCtx, &sql.TxOptions{Isolation: isolation})
	if common.Error(err) {
		return err
	}

	return nil
}

func (sqlDb *SqlDb) Rollback() error {
	err := sqlDb.open()
	if common.Error(err) {
		return err
	}

	sqlDb.txCounter--

	if sqlDb.txCounter > 0 {
		return nil
	}

	err = sqlDb.tx.Rollback()
	if common.Error(err) {
		return err
	}

	sqlDb.txCtxCancel()

	sqlDb.tx = nil
	sqlDb.txCtx = nil
	sqlDb.txCtxCancel = nil

	return nil
}

func (sqlDb *SqlDb) Commit() error {
	err := sqlDb.open()
	if common.Error(err) {
		return err
	}

	sqlDb.txCounter--

	if sqlDb.txCounter > 0 {
		return nil
	}

	err = sqlDb.tx.Commit()
	if common.Error(err) {
		return err
	}

	sqlDb.txCtxCancel()

	sqlDb.tx = nil
	sqlDb.txCtx = nil
	sqlDb.txCtxCancel = nil

	return nil
}

func (sqlDb *SqlDb) Execute(sqlcmd string, args ...any) (int64, error) {
	err := sqlDb.open()
	if common.Error(err) {
		return 0, err
	}

	ctx := context.Background()
	if *FlagDbQueryTimeout != 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(context.Background(), common.MillisecondToDuration(*FlagDbQueryTimeout))
		defer cancel()
	}

	result, err := sqlDb.currentDb().ExecContext(ctx, sqlcmd, args...)
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

func (f Field) String() string {
	if f.IsNull {
		return ""
	}

	return fmt.Sprintf("%v", f.Value)
}

type Resultset struct {
	ColumnNames []string
	ColumnTypes []*sql.ColumnType
	RowCount    int
	Rows        any
}

func cleanFieldname(fieldname string) string {
	return strings.ToUpper(regexFieldName.FindString(fieldname))
}

func (rs *Resultset) FieldByName(row int, fieldName string) Field {
	resultsetValue := reflect.ValueOf(rs.Rows)
	rowValue := resultsetValue.Elem().Index(row)
	colValue := rowValue.FieldByName(cleanFieldname(fieldName))

	return colValue.Interface().(Field)
}

func (rs *Resultset) FieldByIndex(row int, col int) Field {
	resultsetValue := reflect.ValueOf(rs.Rows)
	rowValue := resultsetValue.Elem().Index(row)
	colValue := rowValue.FieldByIndex([]int{col})

	return colValue.Interface().(Field)
}

func (sqlDb *SqlDb) Query(sqlcmd string, args ...any) (*Resultset, error) {
	return sqlDb.query(nil, -1, sqlcmd, args...)
}

func (sqlDb *SqlDb) QueryPaged(fn ResultsetFunc, pageRowCount int, sqlcmd string, args ...any) (int, error) {
	rows, err := sqlDb.query(fn, pageRowCount, sqlcmd, args...)
	if common.Error(err) {
		return 0, err
	}

	return rows.RowCount, nil
}

type ResultsetFunc func(rs *Resultset) error

func (sqlDb *SqlDb) query(fn ResultsetFunc, pageRowCount int, sqlcmd string, args ...any) (*Resultset, error) {
	err := sqlDb.open()
	if common.Error(err) {
		return nil, err
	}

	ctx := context.Background()
	if *FlagDbQueryTimeout != 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(context.Background(), common.MillisecondToDuration(*FlagDbQueryTimeout))
		defer cancel()
	}

	var query *sql.Rows
	err = common.Catch(func() error {
		var err error

		query, err = sqlDb.currentDb().QueryContext(ctx, sqlcmd, args...)

		return err

	})
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
		columnNames[i] = cleanFieldname(columnNames[i])
	}

	columnTypes, err := query.ColumnTypes()
	if common.Error(err) {
		return nil, err
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

	count := 0

	rows := &Resultset{
		ColumnNames: columnNames,
		ColumnTypes: columnTypes,
	}

	hasNext := true

	for {
		recordBuf := bytes.Buffer{}
		recordBuf.WriteString("[")

		rows.RowCount = 0

		for pageRowCount == -1 || (rows.RowCount < pageRowCount) {
			hasNext = query.Next()
			if !hasNext {
				break
			}

			count++
			rows.RowCount++

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
				default:
					s, ok := scanPtrs[i].(*sql.NullString)
					isNull := !(ok && s.Valid)
					recordFieldIsNull.Set(reflect.ValueOf(isNull))
					recordFieldIsNull.Set(reflect.ValueOf(!(ok && s.Valid)))
					if ok && s.Valid {
						recordFieldValue.Set(reflect.ValueOf(s.String))
					}
				}
			}

			ba, err := json.Marshal(record)
			if common.Error(err) {
				return nil, err
			}

			if rows.RowCount > 1 {
				recordBuf.WriteString(",")
			}

			recordBuf.Write(ba)
		}

		recordBuf.WriteString("]")

		rows.Rows = recordStruct.NewSliceOfStructs()

		err = json.Unmarshal(recordBuf.Bytes(), &rows.Rows)
		if common.Error(err) {
			return nil, err
		}

		if rows.RowCount > 0 && fn != nil {
			err := fn(rows)
			if common.Error(err) {
				return nil, err
			}
		}

		if !hasNext {
			break
		}
	}

	rows.RowCount = count

	return rows, nil
}
