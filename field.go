package aorm

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

// Field model field definition
type Field struct {
	*StructField
	IsBlank bool
	Field   reflect.Value
	scaner  Scanner
}

func (this *Field) CallMethodCallbackArgs(name string, object reflect.Value, in []reflect.Value) {
	this.StructField.CallMethodCallbackArgs(name, object, append([]reflect.Value{reflect.ValueOf(this)}, in...))
}

// Call the method callback
func (this *Field) CallMethodCallback(name string, object reflect.Value, in ...reflect.Value) {
	this.CallMethodCallbackArgs(name, object, in)
}

func (this *Field) Scaner(dialect Dialector) Scanner {
	if this.scaner == nil {
		if this.StructField.Assigner != nil {
			this.scaner = this.StructField.Assigner.Scaner(dialect, this.Field)
		} else {
			this.scaner = NewFieldScanner(this)
		}
	}
	return this.scaner
}

// Set set a value to the field
func (this *Field) Set(value interface{}) (err error) {
	if !this.Field.IsValid() {
		return errors.New("field value not valid")
	}

	if !this.Field.CanAddr() {
		return ErrUnaddressable
	}

	reflectValue, ok := value.(reflect.Value)
	if !ok {
		reflectValue = reflect.ValueOf(value)
	}

	fieldValue := this.Field
	if reflectValue.IsValid() {
		if reflectValue.Type().ConvertibleTo(fieldValue.Type()) {
			fieldValue.Set(reflectValue.Convert(fieldValue.Type()))
		} else {
			if fieldValue.Kind() == reflect.Ptr {
				if fieldValue.IsNil() {
					fieldValue.Set(reflect.New(this.Struct.Type.Elem()))
				}
				fieldValue = fieldValue.Elem()
			}

			if reflectValue.Type().ConvertibleTo(fieldValue.Type()) {
				fieldValue.Set(reflectValue.Convert(fieldValue.Type()))
			} else if scanner, ok := fieldValue.Addr().Interface().(sql.Scanner); ok {
				err = scanner.Scan(reflectValue.Interface())
			} else {
				err = fmt.Errorf("could not convert argument of field %s from %s to %s", this.Name, reflectValue.Type(), fieldValue.Type())
			}
		}
	} else {
		this.Field.Set(reflect.Zero(this.Field.Type()))
	}

	this.IsBlank = IsBlank(this.Field)
	return err
}

func (this *Field) ID() (IDValuer, error) {
	return this.StructField.IDOf(this.Field)
}
