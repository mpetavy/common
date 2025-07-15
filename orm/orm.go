package orm

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/fatih/structtag"
	"github.com/mpetavy/common"
	"github.com/mpetavy/common/sqldb"
	"github.com/nsf/jsondiff"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"reflect"
	"strings"
	"time"
)

var (
	ormSlowThreshold = common.SystemFlagInt("orm.slowthreshold", 10000, "Database action slow processing time threshold")
	ormLogLevel      = common.SystemFlagString("orm.log.level", "INFO", "Log level to start logging to database")

	devDbTruncate = common.SystemFlagBool("dev.db.truncate", false, "Database complete table truncate (CAUTION !! data loss)")
)

type ORMDriver interface {
	Dialector() gorm.Dialector
	Synchronized(*sql.DB, func() error) error
}

type ORMSchemaModel struct {
	Model       any
	Schema      any
	CanTruncate bool
}

type Log struct {
	ID        sqldb.FieldInt64  `json:"id" desc:"Unique database ID"`
	CreatedAt sqldb.FieldTime   `json:"createdAt" gorm:"index" desc:"Timestamp this DB record has been created"`
	UpdatedAt sqldb.FieldTime   `json:"updatedAt" gorm:"index" desc:"Timestamp this DB record has been updated"`
	Level     sqldb.FieldString `json:"level" gorm:"index" desc:"level"`
	Source    sqldb.FieldString `json:"source" gorm:"index" desc:"source"`
	Msg       sqldb.FieldString `json:"msg" desc:"msg"`
}

var LogSchema = struct {
	TableName string
	ID        string
	CreatedAt string
	UpdatedAt string
	Level     string
	Source    string
	Msg       string
}{}

type DBInfo struct {
	ID            int       `json:"id" desc:"Unique database ID"`
	CreatedAt     time.Time `json:"createdAt" gorm:"index" desc:"Timestamp this DB record has been created"`
	UpdatedAt     time.Time `json:"updatedAt" gorm:"index" desc:"Timestamp this DB record has been updated the last time"`
	SchemaVersion int       `json:"schemaVersion" desc:"Database schema version"`
}

var DBInfoSchema = struct {
	TableName     string
	ID            string
	CreatedAt     string
	UpdatedAt     string
	SchemaVersion string
}{}

type SetupFunc func(gormDB *gorm.DB) error

type MigrateFunc func(gormDB *gorm.DB, schemaVersion int) error

type ORM struct {
	driver        ORMDriver
	SchemaVersion int
	SchemaModels  []ORMSchemaModel
	Gorm          *gorm.DB
	config        *gorm.Config
}

type gormLogger struct {
	io.Writer
}

func (lw gormLogger) Printf(s string, v ...any) {
	if len(v) > 0 {
		s = fmt.Sprintf(strings.ReplaceAll(s, "\n", ""), v...)
	}

	common.Debug("GORM: %s", s)
}

func UpdateSchema(db *gorm.DB, st *common.StringTable, model any, schema any) (string, error) {
	common.DebugFunc()

	modelStruct, ok := model.(reflect.Value)
	if !ok {
		modelStruct = reflect.Indirect(reflect.ValueOf(model))
	}

	modelType := modelStruct.Type()
	if modelType.Kind() != reflect.Struct {
		return "", fmt.Errorf("model type should be a struct")
	}

	tableName := db.Config.NamingStrategy.TableName(modelType.Name())

	schemaStruct, ok := schema.(reflect.Value)
	if !ok {
		schemaStruct = reflect.Indirect(reflect.ValueOf(schema))
	}

	schemaType := modelStruct.Type()
	if schemaType.Kind() != reflect.Struct {
		return "", fmt.Errorf("model type should be a struct")
	}

	fieldTableName := "TableName"
	field := schemaStruct.FieldByName(fieldTableName)
	if !field.CanSet() {
		return "", fmt.Errorf("field does not exist in schema struct: %s", fieldTableName)
	}

	field.Set(reflect.ValueOf(tableName))

	if modelType.NumField() != schemaType.NumField() {
		return "", fmt.Errorf("model does not have the same number of fields")
	}

	st.AddCols("", "", "", "")
	st.AddCols(fmt.Sprintf("**%s**", tableName), "", "", "")
	st.AddCols("", "", "", "")

	for i := 0; i < modelType.NumField(); i++ {
		fieldName := modelType.Field(i).Name
		dbFieldName := db.Config.NamingStrategy.ColumnName(modelStruct.String(), fieldName)

		fieldTags, err := structtag.Parse(string(modelType.Field(i).Tag))
		if common.Error(err) {
			return "", err
		}

		descTag, err := fieldTags.Get("desc")
		if descTag != nil {
			st.AddCols(tableName, dbFieldName, modelType.Field(i).Type.String(), descTag.Value())
		}

		gormTag, err := fieldTags.Get("gorm")
		if gormTag != nil {
			if strings.HasPrefix(gormTag.Value(), "foreignKey") {
				continue
			}
		}

		_, ok := schemaType.FieldByName(fieldName)
		if !ok {
			return "", fmt.Errorf("field does not exist in struct: %s", fieldName)
		}

		field := schemaStruct.FieldByName(fieldName)

		if !field.CanSet() {
			return "", fmt.Errorf("field does not exist in schema struct: %s", fieldName)
		}

		field.Set(reflect.ValueOf(dbFieldName))
	}

	return tableName, nil
}

func NewORM(driver ORMDriver, schemaVersion int, schemaModels []ORMSchemaModel) (*ORM, error) {
	common.DebugFunc()

	common.StartInfo("Database")

	logLevel := logger.Error
	if common.IsLogVerboseEnabled() {
		logLevel = logger.Silent
	}

	logLevel = logger.Info

	newLogger := logger.New(&gormLogger{},
		logger.Config{
			SlowThreshold:             common.MillisecondToDuration(*ormSlowThreshold), // Slow SQL threshold
			LogLevel:                  logLevel,                                        // Log level
			IgnoreRecordNotFoundError: false,                                           // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      false,                                           // Don't include params in the SQL log
			Colorful:                  false,                                           // Disable color
		},
	)
	newLogger.LogMode(logger.Info)

	orm := &ORM{
		driver:        driver,
		SchemaVersion: schemaVersion,
		SchemaModels:  schemaModels,
		Gorm:          nil,
		config: &gorm.Config{
			Logger:                 newLogger,
			SkipDefaultTransaction: true,
		},
	}

	var err error

	orm.Gorm, err = gorm.Open(driver.Dialector(), orm.config)
	if common.Error(err) {
		return nil, err
	}

	db, err := orm.Gorm.DB()
	if common.Error(err) {
		return nil, err
	}

	err = db.Ping()
	if common.Error(err) {
		return nil, err
	}

	return orm, nil
}

func (orm *ORM) Reset() error {
	return nil
}

func (orm *ORM) Prepare(setupFunc SetupFunc, migrateFunc MigrateFunc) error {
	common.DebugFunc()

	common.Info("Prepare database")

	err := orm.Synchronized(func() error {
		err := orm.migrate(orm.Gorm, migrateFunc)
		if common.Error(err) {
			return err
		}

		if setupFunc != nil {
			err = setupFunc(orm.Gorm)
			if common.Error(err) {
				return err
			}
		}

		return nil
	})
	if common.Error(err) {
		return err
	}

	err = orm.setupLogger()
	if common.Error(err) {
		return err
	}

	return nil
}

func (orm *ORM) migrate(txTransaction *gorm.DB, migrateFunc MigrateFunc) error {
	common.DebugFunc()

	dbInfo := &DBInfo{}

	txTransaction.First(dbInfo)

	common.Debug("Database schema version: %d", dbInfo.SchemaVersion)

	if dbInfo.SchemaVersion > orm.SchemaVersion {
		return fmt.Errorf("Invalid database schema version, found %d but software wants %d", dbInfo.SchemaVersion, orm.SchemaVersion)
	}

	doMigrate := dbInfo.SchemaVersion != orm.SchemaVersion

	if doMigrate {
		common.Info("Migrate database: %d -> %d", dbInfo.SchemaVersion, orm.SchemaVersion)
	}

	if *devDbTruncate {
		common.Info("Truncate database")
	}

	st := common.NewStringTable()
	st.AddCols("Table", "fieldname", "type", "description")

	for i := 0; i < len(orm.SchemaModels); i++ {
		if i > 0 {
			st.AddCols("", "", "", "")
		}

		tableName, err := UpdateSchema(txTransaction, st, orm.SchemaModels[i].Model, orm.SchemaModels[i].Schema)
		if common.Error(err) {
			return err
		}

		if doMigrate {
			err = txTransaction.AutoMigrate(orm.SchemaModels[i].Model)
			if common.Error(err) {
				return err
			}
		}

		if *devDbTruncate && orm.SchemaModels[i].CanTruncate {
			tx := txTransaction.Exec(fmt.Sprintf("delete from %s", tableName))
			if common.Error(tx.Error) {
				return tx.Error
			}

			tx = txTransaction.Exec(fmt.Sprintf("alter sequence %s_id_seq restart with 1", tableName))
			if common.Error(tx.Error) {
				return tx.Error
			}
		}
	}

	common.Debug("Schema:\n" + st.Markdown())

	dbInfo = &DBInfo{}

	tx := txTransaction.FirstOrCreate(dbInfo)
	if common.Error(tx.Error) {
		return tx.Error
	}

	if doMigrate {
		err := orm.Gorm.Transaction(func(txTransaction *gorm.DB) error {
			if migrateFunc != nil {
				for schemaVersion := dbInfo.SchemaVersion; schemaVersion < orm.SchemaVersion; schemaVersion++ {
					err := migrateFunc(txTransaction, schemaVersion)
					if common.Error(err) {
						return err
					}
				}
			}

			dbInfo.SchemaVersion = orm.SchemaVersion

			tx = txTransaction.Save(dbInfo)
			if common.Error(tx.Error) {
				return tx.Error
			}

			return nil
		})
		if common.Error(err) {
			return err
		}

		return nil
	}

	return nil
}

func (orm *ORM) setupLogger() error {
	if *ormLogLevel != "" {
		common.Events.AddListener(common.EventLog{}, func(event common.Event) {
			eventLog := event.(common.EventLog)

			level := common.LevelToIndex(eventLog.Entry.Level)

			if level != -1 && level < common.LevelToIndex(*ormLogLevel) {
				return
			}

			db, err := orm.Gorm.DB()
			if err != nil {
				return
			}

			err = db.Ping()
			if err != nil {
				return
			}

			msg := eventLog.Entry.Msg
			if common.LevelToIndex(eventLog.Entry.Level) >= common.LevelToIndex(common.LevelError) {
				msg = eventLog.Entry.StacktraceMsg
			}

			logging := &Log{
				CreatedAt: sqldb.NewFieldTime(eventLog.Entry.Time),
				UpdatedAt: sqldb.NewFieldTime(eventLog.Entry.Time),
				Level:     sqldb.NewFieldString(eventLog.Entry.Level),
				Source:    sqldb.NewFieldString(eventLog.Entry.Source),
				Msg:       sqldb.NewFieldString(msg),
			}

			tx := orm.Gorm.Create(logging)
			if common.Error(tx.Error) {
				common.Error(tx.Error)
			}
		})
	}

	return nil
}

func (orm *ORM) Close() error {
	common.StopInfo("Database")

	return nil
}

func (orm *ORM) Health() error {
	common.DebugFunc()

	db, err := orm.Gorm.DB()
	if common.Error(err) {
		return err
	}

	err = db.Ping()
	if common.Error(err) {
		return err
	}

	return nil
}

func (orm *ORM) Synchronized(fn func() error) error {
	db, err := orm.Gorm.DB()
	if common.Error(err) {
		return err
	}

	err = orm.driver.Synchronized(db, fn)
	if common.Error(err) {
		return err
	}

	return nil
}

func (orm *ORM) VerifyCfgChanged(cfg any) error {
	cfgJson, err := json.MarshalIndent(cfg, "", "    ")
	if common.Error(err) {
		return err
	}

	log := &Log{}

	tx := orm.Gorm.Last(log, fmt.Sprintf("%s = ?", LogSchema.Level), "CFG")
	hasChanged := tx.RowsAffected == 0

	if !hasChanged {
		diff, ba, err := common.JSONCompare([]byte(log.Msg.String()), cfgJson, jsondiff.DefaultJSONOptions(), func(keyPath string, value any, depth int) bool {
			for _, wildcard := range []string{"status*", "cfg*", "build", "git", "flags*"} {
				b, _ := common.EqualsWildcard(keyPath, wildcard)
				if b {
					return false
				}
			}

			return true
		})
		if common.Error(err) {
			return err
		}

		hasChanged = diff != jsondiff.FullMatch

		if hasChanged {
			common.Events.Emit(common.EventLog{
				Entry: common.NewLogEntry("CFG-CHANGED", string(ba), common.GetRuntimeInfo(0)),
			}, false)
		}
	}

	if hasChanged {
		common.Events.Emit(common.EventLog{
			Entry: common.NewLogEntry("CFG", string(cfgJson), common.GetRuntimeInfo(0)),
		}, false)
	}

	return nil
}
