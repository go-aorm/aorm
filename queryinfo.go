package aorm

import (
	"database/sql"
	"database/sql/driver"
)

// Represents Query with args Info
type QueryInfo struct {
	Query
	argsName []string
}

func NewQueryInfo(q Query, varBinder func(i int) string) *QueryInfo {
	qe := &QueryInfo{Query: q, argsName: make([]string, len(q.Args), len(q.Args))}
	for i := range qe.Query.Args {
		qe.argsName[i] = varBinder(i + 1)
	}
	return qe
}

func (e *QueryInfo) Sql() string {
	return e.Query.Query
}

func (e *QueryInfo) Args() []interface{} {
	return e.Query.Args
}

func (e *QueryInfo) ArgsName() []string {
	return e.argsName
}

func (e *QueryInfo) EachArgs(cb func(i int, name string, value interface{})) {
	for i := range e.Query.Args {
		cb(i, e.argsName[i], e.Query.Args[i])
	}
}

func (e *QueryInfo) String() string {
	var args = make([]interface{}, len(e.Query.Args), len(e.Query.Args))
	e.EachArgs(func(i int, name string, value interface{}) {
		if vlr, ok := value.(driver.Valuer); ok {
			if v, err := vlr.Value(); err == nil {
				value = v
			}
		}
		args[i] = sql.Named(e.argsName[i], value)
	})
	return Query{e.Query.Query, args}.String()
}
