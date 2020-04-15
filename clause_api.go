package aorm

type (
	SqlClauser interface {
		SqlClause(scope *Scope) (query string)
	}
)
