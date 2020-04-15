package aorm

import (
	"fmt"

	"github.com/pkg/errors"
)

func (scope *Scope) Clause(clause interface{}) string {
	if sql, ok := scope.ClauseToSql(clause); ok {
		return sql
	}
	scope.Err(fmt.Errorf("bad clause type: %T", clause))
	return ""
}

func (scope *Scope) ClauseToSql(clause interface{}) (string, bool) {
	switch t := clause.(type) {
	case string:
		return scope.quoteIfPossible(t), true
	case Query:
		if q, err := t.Build(scope); err != nil {
			scope.Err(errors.Wrap(err, "build Query"))
			return "", true
		} else {
			return q, true
		}
	case *Query:
		if q, err := t.Build(scope); err != nil {
			scope.Err(errors.Wrap(err, "build *Query"))
			return "", true
		} else {
			return q, true
		}
	case SqlClauser:
		return scope.quoteIfPossible(t.SqlClause(scope)), true
	case Clauser:
		if q, err := t.Clause(scope).Build(scope); err != nil {
			scope.Err(errors.Wrap(err, "build Clauser"))
			return "", true
		} else {
			return q, true
		}
	case WhereClauser:
		if q, err := t.WhereClause(scope).Build(scope); err != nil {
			scope.Err(errors.Wrap(err, "build Clauser"))
			return "", true
		} else {
			return q, true
		}
	default:
		return "", false
	}
}
