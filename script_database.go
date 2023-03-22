package common

func (engine *ScriptEngine) EnableDatabase() error {
	db := &Database{}

	obj := engine.VM.NewObject()

	err := obj.Set("drivers", DatabaseDrivers())
	if Error(err) {
		return err
	}

	err = obj.Set("init", db.Init)
	if Error(err) {
		return err
	}

	err = obj.Set("begin", db.Begin)
	if Error(err) {
		return err
	}

	err = obj.Set("close", db.Close)
	if Error(err) {
		return err
	}

	err = obj.Set("commit", db.Commit)
	if Error(err) {
		return err
	}

	err = obj.Set("execute", db.Execute)
	if Error(err) {
		return err
	}

	err = obj.Set("open", db.Open)
	if Error(err) {
		return err
	}

	err = obj.Set("query", db.Query)
	if Error(err) {
		return err
	}

	err = obj.Set("rollback", db.Rollback)
	if Error(err) {
		return err
	}

	err = engine.VM.Set("database", obj)
	if Error(err) {
		return err
	}

	return nil
}
