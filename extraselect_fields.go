package aorm

import (
	"reflect"
)

type extraSelectsFields struct {
	Items  []*extraSelectField
	Fields []*StructField
	Types  []reflect.Type
	Ptrs   []bool
	Size   int
}

func (es *extraSelectsFields) NewValues() []interface{} {
	r := make([]interface{}, es.Size)
	for i, typ := range es.Types {
		scaner := NewValueScanner(typ, es.Ptrs[i])
		scaner.StructField = es.Fields[i]
		r[i] = scaner
	}
	return r
}
func (es *extraSelectsFields) Add(key string, value interface{}, fields []*StructField, callback func(scope *Scope, record interface{}), query interface{}, args []interface{}) *extraSelectField {
	types := make([]reflect.Type, len(fields))
	ptrs := make([]bool, len(fields))

	for i, field := range fields {
		typ := field.Struct.Type
		for typ.Kind() == reflect.Ptr {
			ptrs[i] = true
			typ = typ.Elem()
		}
		types[i] = typ
	}

	s := &extraSelectField{Clause{query, args}, key, value, fields, ptrs, callback, len(fields)}
	es.Fields = append(es.Fields, fields...)
	es.Types = append(es.Types, types...)
	es.Ptrs = append(es.Ptrs, ptrs...)
	es.Items = append(es.Items, s)
	es.Size += s.Size
	return s
}

func (ess *extraSelectsFields) SetValues(scope *Scope, result interface{}, values []interface{}) {
	for _, es := range ess.Items {
		es.SetValues(scope, result, values[0:len(es.Fields)])
		values = values[es.Size:]
	}
}

type extraSelectField struct {
	Clause
	key      string
	Value    interface{}
	Fields   []*StructField
	Ptrs     []bool
	callback func(scope *Scope, record interface{})
	Size     int
}

type ExtraSelectFieldsSetter func(result interface{}, values []interface{}, set func(model *ModelStruct, result interface{}, low, hight int) interface{})

func (es *extraSelectField) setValues(scope *Scope, record interface{}, values []interface{}, fields []*Field, ptrs []bool) {
	scopeValue := reflect.ValueOf(scope)
	for i, field := range fields {
		fieldScanner := values[i].(*ValueScanner)
		fieldValue := reflect.ValueOf(fieldScanner.Data)
		if ptrs[i] {
			field.Field.Set(fieldValue)
		} else {
			field.Field.Set(fieldValue.Elem())
		}

		// if not is nill and if calbacks enabled for field type
		if StructFieldMethodCallbacks.IsEnabledFieldType(field.Field.Type()) {
			if !isNil(field.Field) {
				reflectValue := field.Field.Addr()
				field.CallMethodCallback("AfterScan", reflectValue, scopeValue)
			}
		}
	}

	if es.callback != nil {
		es.callback(scope, record)
	}
}

func (es *extraSelectField) SetValues(scope *Scope, record interface{}, values []interface{}) {
	if setter, ok := es.Value.(ExtraSelectFieldsSetter); ok {
		setter(record, values, func(model *ModelStruct, result interface{}, low, hight int) interface{} {
			var newScope *Scope
			switch rt := result.(type) {
			case reflect.Value:
				newScope = scope.New(reflect.Indirect(rt).Addr().Interface())
			case reflect.Type:
				newScope = scope.New(reflect.New(rt).Addr())
			default:
				newScope = scope.New(reflect.New(reflect.Indirect(reflect.ValueOf(result)).Type()).Interface())
			}
			newScope.modelStruct = model
			if hight <= 0 {
				hight = len(values)
			}
			fields := newScope.ValuesFields(es.Fields[low:hight])
			es.setValues(newScope, record, values[low:hight], fields, es.Ptrs[low:hight])
			return newScope.Value
		})
	} else {
		newScope := scope.New(reflect.New(reflect.Indirect(reflect.ValueOf(es.Value)).Type()).Interface())
		fields := make([]*Field, len(es.Fields))
		for i, rfield := range es.Fields {
			fields[i] = newScope.Instance().FieldsMap[rfield.Name]
		}
		es.setValues(newScope, record, values, fields, es.Ptrs)
	}
}
