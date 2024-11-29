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

type SqlDB struct {
	Driver            string
	DSN               string
	Isolation         sql.IsolationLevel
	Conn              *sql.DB
	txCounter         int
	tx                *sql.Tx
	txCtx             context.Context
	txCtxCancel       context.CancelFunc
	lastUsage         time.Time
	QueryTimeout      time.Duration
	RevalidateTimeout time.Duration
}

var (
	revalidateTimeout = common.SystemFlagInt("sqldb.revalidatetimeout", 5*60*1000, "Timeout after db connection is revalidated")
	queryTimeout      = common.SystemFlagInt("sqldb.querytimeout", 30*1000, "Timeout for db queries")

	regexFieldName = regexp.MustCompile("([\\w\\d_]+)")
)

type dbintf interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func NewSqlDB(driver, dsn string) (*SqlDB, error) {
	db := &SqlDB{}

	err := db.Init(driver, dsn)
	if common.Error(err) {
		return nil, err
	}

	return db, nil
}

func (sqlDb *SqlDB) currentDb() dbintf {
	if sqlDb.tx != nil {
		return sqlDb.tx
	} else {
		return sqlDb.Conn
	}
}

func (sqlDb *SqlDB) Revalidate(force bool) error {
	fmt.Printf("-----------------\n")
	fmt.Printf("%v\n", force)
	fmt.Printf("%v\n", time.Now())
	fmt.Printf("%v\n", sqlDb.lastUsage)

	if !force && sqlDb.lastUsage.Add(sqlDb.RevalidateTimeout).After(time.Now()) {
		sqlDb.lastUsage = time.Now()

		return nil
	}

	for try := 0; ; try++ {
		ctx := context.Background()
		if sqlDb.QueryTimeout != 0 {
			var cancel context.CancelFunc

			ctx, cancel = context.WithTimeout(context.Background(), sqlDb.QueryTimeout)
			defer cancel()
		}

		err := sqlDb.Conn.PingContext(ctx)
		if !common.Error(err) {
			sqlDb.lastUsage = time.Now()

			return nil
		}
		if try > 0 {
			return err
		}

		common.IgnoreError(sqlDb.Close())

		err = sqlDb.Open()
		if common.Error(err) {
			return err
		}
	}
}

func (sqlDb *SqlDB) Init(driver, dsn string) error {
	sqlDb.Driver = driver
	sqlDb.DSN = dsn
	sqlDb.QueryTimeout = common.MillisecondToDuration(*queryTimeout)
	sqlDb.RevalidateTimeout = common.MillisecondToDuration(*revalidateTimeout)
	sqlDb.Isolation = sql.LevelReadCommitted

	return nil
}

func (sqlDb *SqlDB) Open() error {
	db, err := sql.Open(sqlDb.Driver, sqlDb.DSN)
	if common.Error(err) {
		return err
	}

	sqlDb.Conn = db

	err = sqlDb.Revalidate(true)
	if common.Error(err) {
		return err
	}

	return nil
}

func (sqlDb *SqlDB) Close() error {
	err := sqlDb.Conn.Close()
	if common.Error(err) {
		return err
	}

	sqlDb.Conn = nil

	return nil
}

func (sqlDb *SqlDB) Begin() error {
	err := sqlDb.Revalidate(false)
	if common.Error(err) {
		return err
	}

	sqlDb.txCounter++

	if sqlDb.txCounter > 1 {
		return nil
	}

	sqlDb.txCtx, sqlDb.txCtxCancel = context.WithCancel(context.Background())

	sqlDb.tx, err = sqlDb.Conn.BeginTx(sqlDb.txCtx, &sql.TxOptions{Isolation: sqlDb.Isolation})
	if common.Error(err) {
		return err
	}

	return nil
}

func (sqlDb *SqlDB) Rollback() error {
	err := sqlDb.Revalidate(false)
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

func (sqlDb *SqlDB) Commit() error {
	err := sqlDb.Revalidate(false)
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

func (sqlDb *SqlDB) Execute(sqlcmd string, args ...any) (int64, error) {
	err := sqlDb.Revalidate(false)
	if common.Error(err) {
		return 0, err
	}

	ctx := context.Background()
	if sqlDb.QueryTimeout != 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(context.Background(), sqlDb.QueryTimeout)
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

func (f *Field) String() string {
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

func (sqlDb *SqlDB) Query(sqlcmd string, args ...any) (*Resultset, error) {
	err := sqlDb.Revalidate(false)
	if common.Error(err) {
		return nil, err
	}

	ctx := context.Background()
	if sqlDb.QueryTimeout != 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(context.Background(), sqlDb.QueryTimeout)
		defer cancel()
	}

	query, err := sqlDb.currentDb().QueryContext(ctx, sqlcmd, args...)
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

	rows := &Resultset{
		ColumnNames: columnNames,
		ColumnTypes: columnTypes,
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

	rows.Rows = recordStruct.NewSliceOfStructs()

	if recordBuf.Len() > 0 {
		recordBuf.WriteString("]")

		err = json.Unmarshal(recordBuf.Bytes(), &rows.Rows)
		if common.Error(err) {
			return nil, err
		}
	}

	return rows, nil
}
