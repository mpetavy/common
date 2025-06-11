package scripting

import (
	"database/sql"
	"github.com/mpetavy/common"
	"github.com/mpetavy/common/sqldb"
)

func EnableDatabase(engine *ScriptEngine) error {
	sqlDb := &sqldb.SqlDb{}

	obj := engine.VM.NewObject()

	err := obj.Set("drivers", sql.Drivers())
	if common.Error(err) {
		return err
	}

	err = obj.Set("init", sqlDb.Init)
	if common.Error(err) {
		return err
	}

	err = obj.Set("begin", sqlDb.Begin)
	if common.Error(err) {
		return err
	}

	err = obj.Set("close", sqlDb.Close)
	if common.Error(err) {
		return err
	}

	err = obj.Set("commit", sqlDb.Commit)
	if common.Error(err) {
		return err
	}

	err = obj.Set("execute", sqlDb.Execute)
	if common.Error(err) {
		return err
	}

	err = obj.Set("open", sqlDb.Open)
	if common.Error(err) {
		return err
	}

	err = obj.Set("query", sqlDb.Query)
	if common.Error(err) {
		return err
	}

	err = obj.Set("rollback", sqlDb.Rollback)
	if common.Error(err) {
		return err
	}

	err = engine.VM.Set("database", obj)
	if common.Error(err) {
		return err
	}

	return nil
}
