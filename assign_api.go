package aorm

import (
	"database/sql/driver"
	"reflect"
)

type (
	Assigner interface {
		Valuer(dialect Dialector, value interface{}) driver.Valuer
		Scaner(dialect Dialector, dest reflect.Value) Scanner
		SQLType(dialect Dialector) string
		Type() reflect.Type
	}

	AssignArgBinder interface {
		Assigner
		ArgBinder
	}

	SQLSizer interface {
		SQLSize(dialect Dialector) int
	}
)
