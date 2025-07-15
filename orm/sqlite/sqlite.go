package sqlite

import (
	"database/sql"
	"flag"
	"github.com/mpetavy/common"
	"github.com/mpetavy/common/orm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	sqliteFile *string
)

func init() {
	common.Events.AddListener(common.EventInit{}, func(event common.Event) {
		sqliteFile = flag.String("sqlite.file", common.AppFilename(".db"), "Database file")
	})
}

type SqliteDriver struct {
	orm.ORMDriver
	dialector gorm.Dialector
}

func NewDriver() (*SqliteDriver, error) {
	driver := &SqliteDriver{}

	driver.dialector = sqlite.Open(*sqliteFile)

	return driver, nil
}

func (driver *SqliteDriver) Dialector() gorm.Dialector {
	return driver.dialector
}

func (driver *SqliteDriver) Synchronized(db *sql.DB, fn func() error) error {
	err := fn()
	if common.Error(err) {
		return err
	}

	return nil
}
