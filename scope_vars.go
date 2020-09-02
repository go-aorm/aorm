package aorm

import "reflect"

type RawSqlArg string

// AddToVars add value as sql's vars, used to prevent SQL injection
func (scope *Scope) AddToVars(value interface{}) (replacement string) {
	if raw, ok := value.(RawSqlArg); ok {
		return string(raw)
	}

	if _, ok := value.(string); !ok {
		if sql, ok := scope.ClauseToSql(value); ok {
			return sql
		}
	}

	typ := reflect.TypeOf(value)
	assigner := scope.db.GetAssigner(typ)

	replacement = "?"
	if _, skipBindVar := scope.InstanceGet("skip_bindvar"); !skipBindVar {
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
