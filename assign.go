package aorm

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
)

type AssignerRegistrator struct {
	data map[reflect.Type]Assigner
}

func (a *AssignerRegistrator) Register(assigners ...Assigner) {
	if a.data == nil {
		a.data = map[reflect.Type]Assigner{}
	}
	for _, assigner := range assigners {
		a.data[assigner.Type()] = assigner
	}
}

func (a *AssignerRegistrator) Get(typ reflect.Type) (assigner Assigner) {
	if a.data != nil {
		assigner = a.data[typ]
	}
	return
}

func (a *AssignerRegistrator) Of(value interface{}) (assigner Assigner) {
	if assigner, ok := value.(Assigner); ok {
		return assigner
	}
	reflectType := reflect.TypeOf(value)
	for reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}
	return a.Get(reflectType)
}

func (a *AssignerRegistrator) Has(typ reflect.Type) (ok bool) {
	if a.data != nil {
		_, ok = a.data[typ]
	}
	return
}

func (a *AssignerRegistrator) ApplyToDialect(dialect Dialector) {
	if a.data != nil {
		for _, assigner := range a.data {
			dialect.RegisterAssigner(assigner)
		}
	}
}

type SimpleAssigner struct {
	ValuerFunc  func(dialect Dialector, value interface{}) driver.Valuer
	ScanerFunc  func(dialect Dialector, dest reflect.Value) Scanner
	Typ         reflect.Type
	SQLtype     string
	SQLtypeFunc func(dialect Dialector) string
}

func (assigner *SimpleAssigner) Type() reflect.Type {
	return assigner.Typ
}

func (assigner *SimpleAssigner) Valuer(dialect Dialector, value interface{}) driver.Valuer {
	return assigner.ValuerFunc(dialect, value)
}

func (assigner *SimpleAssigner) Scaner(dialect Dialector, dest reflect.Value) Scanner {
	return assigner.ScanerFunc(dialect, dest)
}

func (assigner *SimpleAssigner) SQLType(dialect Dialector) string {
	if assigner.SQLtypeFunc == nil {
		return assigner.SQLtype
	}
	return assigner.SQLtypeFunc(dialect)
}

type valuerFunc func() (driver.Value, error)

func (v valuerFunc) Value() (driver.Value, error) {
	return v()
}

func ValuerFunc(f func() (driver.Value, error)) driver.Valuer {
	return valuerFunc(f)
}

type Scanner = sql.Scanner

type scannerFunc func(src interface{}) error

func (this scannerFunc) Scan(src interface{}) error {
	return this(src)
}

func ScannerFunc(f func(src interface{}) error) Scanner {
	return scannerFunc(f)
}
