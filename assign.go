package aorm

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
)

type Assigner interface {
	Valuer(dialect Dialect, value interface{}) driver.Valuer
	Scaner(dialect Dialect, dest reflect.Value) Scanner
	SQLType(dialect Dialect) string
	Type() reflect.Type
}

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

func (a *AssignerRegistrator) GetAssigner(typ reflect.Type) (assigner Assigner) {
	if a.data != nil {
		assigner = a.data[typ]
	}
	return
}

func (a *AssignerRegistrator) HasAssigner(typ reflect.Type) (ok bool) {
	if a.data != nil {
		_, ok = a.data[typ]
	}
	return
}

func (a *AssignerRegistrator) ApplyToDialect(dialect Dialect) {
	if a.data != nil {
		for _, assigner := range a.data {
			dialect.RegisterAssigner(assigner)
		}
	}
}

func (a *AssignerRegistrator) ApplyToDB(db *DB) *DB {
	if a.data != nil {
		var assigners []Assigner
		for _, assigner := range a.data {
			assigners = append(assigners, assigner)
		}
		return db.RegisterAssigner(assigners...)
	}
	return db
}

type SimpleAssigner struct {
	ValuerFunc  func(dialect Dialect, value interface{}) driver.Valuer
	ScanerFunc  func(dialect Dialect, dest reflect.Value) Scanner
	Typ         reflect.Type
	SQLtype     string
	SQLtypeFunc func(dialect Dialect) string
}

func (assigner *SimpleAssigner) Type() reflect.Type {
	return assigner.Typ
}

func (assigner *SimpleAssigner) Valuer(dialect Dialect, value interface{}) driver.Valuer {
	return assigner.ValuerFunc(dialect, value)
}

func (assigner *SimpleAssigner) Scaner(dialect Dialect, dest reflect.Value) Scanner {
	return assigner.ScanerFunc(dialect, dest)
}

func (assigner *SimpleAssigner) SQLType(dialect Dialect) string {
	if assigner.SQLtypeFunc == nil {
		return assigner.SQLtype
	}
	return assigner.SQLtypeFunc(dialect)
}

type ValuerFunc func() (driver.Value, error)

func (v ValuerFunc) Value() (driver.Value, error) {
	return v()
}

type Scanner interface {
	sql.Scanner
	IsPtr() bool
}

type ScannerFunc struct {
	Ptr  bool
	Func func(src interface{}) error
}

func (sf ScannerFunc) IsPtr() bool {
	return sf.Ptr
}

func (s ScannerFunc) Scan(src interface{}) error {
	return s.Func(src)
}
