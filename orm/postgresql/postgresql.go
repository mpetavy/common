package postgresql

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"github.com/mpetavy/common"
	"github.com/mpetavy/common/orm"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	pgHost     = flag.String("pg.host", "", "Database host")
	pgPort     = flag.Int("pg.port", 0, "Database port")
	pgName     = flag.String("pg.dbname", "", "Database name")
	pgUsername = flag.String("pg.username", "", "Database username")
	pgPassword = flag.String("pg.password", "", "Database password")
	pgSslMode  = flag.String("pg.sslmode", "disable", "Database sslmode")
	pgTimezone = flag.String("pg.timezone", "UTC", "Database timezone")
)

type PostgresqlDriver struct {
	orm.ORMDriver
	dialector gorm.Dialector
}

func NewDriver() (*PostgresqlDriver, error) {
	driver := &PostgresqlDriver{}

	dsn := fmt.Sprintf("dbname=%s host=%s port=%d user=%s password=%s sslmode=%s timeZone=%s", *pgName, *pgHost, *pgPort, *pgUsername, *pgPassword, *pgSslMode, *pgTimezone)

	driver.dialector = postgres.Open(dsn)

	return driver, nil
}

func (driver *PostgresqlDriver) Dialector() gorm.Dialector {
	return driver.dialector
}

func (driver *PostgresqlDriver) Synchronized(conn *sql.DB, fn func() error) error {
	lockId := 999111

	_, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_lock($1)", lockId)
	if common.Error(err) {
		return err
	}

	defer func() {
		_, err := conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock($1)", lockId)
		common.Error(err)
	}()

	err = fn()
	if common.Error(err) {
		return err
	}

	return nil
}
