package aorm

const dryRun = "aorm:dry_run"

type ExecState struct {
	Query
	Value interface{}
}

// DryRun perform a trial run with no changes
func (s *DB) DryRun() *DB {
	return s.Set(dryRun, true)
}

func (scope *Scope) checkDryRun() bool {
	if v, ok := scope.db.values[dryRun]; ok && v.(bool) {
		return true
	}
	return false
}

func (scope *Scope) checkDryRunCallback(f func() error) error {
	if !scope.checkDryRun() {
		return f()
	}
	return nil
}
