package aorm

import (
	"database/sql"
	"reflect"
)

func NewFieldScanner(field *Field) *ValueScanner {
	typ, value := field.Struct.Type, field.Field
	s := &ValueScanner{
		Field: field,
		Typ:   typ,
		Set: func(f *ValueScanner) {
			if !f.IsPtr {
				reflectValue := reflect.ValueOf(f.Data).Elem().Elem()
				if reflectValue.IsValid() {
					value.Set(reflectValue)
				}
			}
		}}

	if value.Kind() == reflect.Ptr {
		s.IsPtr = true
		s.Data = value.Addr().Interface()
	} else {
		reflectValue := reflect.New(reflect.PtrTo(typ))
		reflectValue.Elem().Set(value.Addr())
		s.Data = reflectValue.Interface()
	}
	return s
}

func NewValueScanner(typ reflect.Type) *ValueScanner {
	s := &ValueScanner{Typ: typ}
	s.Data = reflect.New(typ).Interface()
	return s
}

type ValueScanner struct {
	Field   *Field
	Typ     reflect.Type
	Data    interface{}
	NotNil  bool
	IsValid bool
	IsPtr   bool
	Set     func(f *ValueScanner)
}

func (f *ValueScanner) IsNil() bool {
	return !f.NotNil
}

func (f *ValueScanner) Scan(src interface{}) error {
	f.NotNil = src != nil
	if scan, ok := f.Data.(sql.Scanner); ok {
		return scan.Scan(src)
	}
	if src != nil {
		err := convertAssign(f.Data, src)
		if err == nil && f.Set != nil {
			f.Set(f)
		}
		return err
	}
	return nil
}
