package types

import (
	"database/sql/driver"
	"reflect"

	"github.com/moisespsena-go/aorm"
)

type (
	Money float64

	MoneyAssigner struct {
	}
)

func init() {
	aorm.Register(MoneyAssigner{})
}

func (MoneyAssigner) Select(field *aorm.StructField, scope *aorm.Scope, table string) aorm.Query {
	switch scope.Dialect().GetName() {
	case "postgres":
		return aorm.Query{Query: table + field.DBName + "::NUMERIC::FLOAT8"}
	}
	return aorm.Query{Query: table + field.DBName}
}

func (MoneyAssigner) SelectWrap(_ *aorm.StructField, scope *aorm.Scope, expr string) aorm.Query {
	switch scope.Dialect().GetName() {
	case "postgres":
		return aorm.Query{Query: expr + "::NUMERIC::FLOAT8"}
	}
	return aorm.Query{Query: expr}
}

func (MoneyAssigner) DbBindVar(dialect aorm.Dialector, argVar string) string {
	switch dialect.GetName() {
	case "postgres":
		return argVar + "::NUMERIC::MONEY"
	}
	return argVar
}

func (MoneyAssigner) Valuer(_ aorm.Dialector, value interface{}) driver.Valuer {
	return aorm.ValuerFunc(func() (driver.Value, error) {
		return float64(value.(Money)), nil
	})
}

func (MoneyAssigner) Scaner(_ aorm.Dialector, dest reflect.Value) aorm.Scanner {
	return aorm.ScannerFunc(func(src interface{}) (err error) {
		if src == nil {
			dest.SetFloat(0)
			return
		}
		var v float64
		if err = aorm.SqlConvertAssign(&v, src); err != nil {
			return
		}
		dest.SetFloat(v)
		return
	})
}

func (MoneyAssigner) SQLType(dialect aorm.Dialector) string {
	switch dialect.GetName() {
	case "postgres":
		return "MONEY"
	}
	return "NUMERIC(20,2)"
}

func (MoneyAssigner) SQLSize(_ aorm.Dialector) int {
	return 0
}

func (MoneyAssigner) Type() reflect.Type {
	return reflect.TypeOf(Money(0))
}
