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
	}

	if value.Kind() == reflect.Ptr {
		s.Ptr = true
		s.Data = value.Interface()
		s.MakePtr = func() interface{} {
			return reflect.New(typ.Elem()).Interface()
		}
		s.Set = func(f *ValueScanner) {
			value.Set(reflect.ValueOf(f.Data))
		}
	} else {
		s.Set = func(f *ValueScanner) {
			reflectValue := reflect.ValueOf(f.Data)
			if reflectValue.Kind() == reflect.Struct {
				reflectValue = reflectValue.Elem()
			}
			if reflectValue.IsValid() {
				value.Set(reflectValue.Elem())
			}
		}
		if value.Kind() == reflect.Struct {
			s.Data = value.Addr().Interface()
		} else {
			reflectValue := reflect.New(reflect.PtrTo(typ))
			reflectValue.Elem().Set(value.Addr())
			s.Data = reflectValue.Elem().Interface()
		}
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
	Ptr     bool
	Set     func(f *ValueScanner)
	MakePtr func() interface{}
}

func (f *ValueScanner) IsNil() bool {
	return !f.NotNil
}

func (f *ValueScanner) IsPtr() bool {
	return f.Ptr
}

func (f *ValueScanner) Scan(src interface{}) error {
	f.NotNil = src != nil
	if scan, ok := f.Data.(sql.Scanner); ok {
		return scan.Scan(src)
	}
	if src != nil {
		if f.Ptr && f.MakePtr != nil {
			f.Data = f.MakePtr()
		}
		var err error
		err = convertAssign(f.Data, src)
		if err == nil && f.Set != nil {
			f.Set(f)
		}
		return err
	}
	return nil
}
