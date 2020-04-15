package aorm

import "reflect"

// AddToVars add value as sql's vars, used to prevent SQL injection
func (scope *Scope) AddToVars(value interface{}) (replacement string) {
	_, skipBindVar := scope.InstanceGet("skip_bindvar")

	if _, ok := value.(string); !ok {
		if sql, ok := scope.ClauseToSql(value); ok {
			return sql
		}
	}

	typ := reflect.TypeOf(value)
	assigner := scope.db.GetAssigner(typ)

	replacement = "?"
	if !skipBindVar {
		replacement = scope.db.dialect.BindVar(len(scope.Query.Args) + 1)
	}

	if binder, ok := value.(ArgBinder); ok {
		replacement = binder.DbBindVar(scope.db.dialect, replacement)
	} else if assigner != nil {
		if binder, ok := assigner.(ArgBinder); ok {
			replacement = binder.DbBindVar(scope.db.dialect, replacement)
		}
		value = assigner.Valuer(scope.db.dialect, value)
	}

	scope.Query.Args = append(scope.Query.Args, value)

	return replacement
}
