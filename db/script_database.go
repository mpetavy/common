package db

import "github.com/mpetavy/common"

func EnableDatabase(engine *common.ScriptEngine) error {
	db := &Database{}

	obj := engine.VM.NewObject()

	err := obj.Set("drivers", DatabaseDrivers())
	if common.Error(err) {
		return err
	}

	err = obj.Set("init", db.Init)
	if common.Error(err) {
		return err
	}

	err = obj.Set("begin", db.Begin)
	if common.Error(err) {
		return err
	}

	err = obj.Set("close", db.Close)
	if common.Error(err) {
		return err
	}

	err = obj.Set("commit", db.Commit)
	if common.Error(err) {
		return err
	}

	err = obj.Set("execute", db.Execute)
	if common.Error(err) {
		return err
	}

	err = obj.Set("open", db.Open)
	if common.Error(err) {
		return err
	}

	err = obj.Set("query", db.Query)
	if common.Error(err) {
		return err
	}

	err = obj.Set("rollback", db.Rollback)
	if common.Error(err) {
		return err
	}

	err = engine.VM.Set("database", obj)
	if common.Error(err) {
		return err
	}

	return nil
}
