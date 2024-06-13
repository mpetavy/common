package scripting

import (
	"github.com/mpetavy/common"
	"github.com/mpetavy/common/sqldb"
)

func EnableDatabase(engine *ScriptEngine) error {
	database := &sqldb.Database{}

	obj := engine.VM.NewObject()

	err := obj.Set("drivers", sqldb.DatabaseDrivers())
	if common.Error(err) {
		return err
	}

	err = obj.Set("init", database.Init)
	if common.Error(err) {
		return err
	}

	err = obj.Set("begin", database.Begin)
	if common.Error(err) {
		return err
	}

	err = obj.Set("close", database.Close)
	if common.Error(err) {
		return err
	}

	err = obj.Set("commit", database.Commit)
	if common.Error(err) {
		return err
	}

	err = obj.Set("execute", database.Execute)
	if common.Error(err) {
		return err
	}

	err = obj.Set("open", database.Open)
	if common.Error(err) {
		return err
	}

	err = obj.Set("query", database.Query)
	if common.Error(err) {
		return err
	}

	err = obj.Set("rollback", database.Rollback)
	if common.Error(err) {
		return err
	}

	err = engine.VM.Set("database", obj)
	if common.Error(err) {
		return err
	}

	return nil
}
