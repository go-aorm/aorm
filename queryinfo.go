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
	return Query{e.Query.Query, NamedStringerArgs(e)}.String()
}

func NamedStringerArgs(qi *QueryInfo) (args []interface{}) {
	args = make([]interface{}, len(qi.Query.Args), len(qi.Query.Args))
	qi.EachArgs(func(i int, name string, value interface{}) {
		switch t := value.(type) {
		case ProtectedStringer:
			value = HiddenStringerValue
		case driver.Valuer:
			if v, err := t.Value(); err == nil {
				value = v
			}
		}
		args[i] = sql.Named(qi.argsName[i], value)
	})
	return
}
