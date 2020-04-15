package aorm

import (
	"fmt"
	"reflect"
)

type Instance struct {
	Struct       *ModelStruct
	ReflectValue reflect.Value
	Value        interface{}
	FieldsMap    map[string]*Field
	Fields       []*Field
	Primary      []*Field
}

// PrimaryField return main primary field, if defined more that one primary fields, will return the one having column name `id` or the first one
func (this *Instance) PrimaryField() *Field {
	if len(this.Primary) > 0 {
		for _, f := range this.Primary {
			if f.DBName == "id" {
				return f
			}
		}
		return this.Primary[0]
	}
	return nil
}

func (this *Instance) RelatedFields() []*Field {
	fields := make([]*Field, len(this.Struct.RelatedFields))
	for i, struc := range this.Struct.RelatedFields {
		fields[i] = this.FieldsMap[struc.Name]
	}
	return fields
}

// FieldByName find `aorm.Field` with field name or db name
func (this *Instance) FieldByName(name string) (field *Field, ok bool) {
	if field, ok = this.FieldsMap[name]; ok {
		return
	}
	var (
		dbName           = ToDBName(name)
		mostMatchedField *Field
	)
	for _, field := range this.Fields {
		if field.DBName == name {
			return field, true
		}
		if field.DBName == dbName {
			mostMatchedField = field
		}
	}
	return mostMatchedField, mostMatchedField != nil
}

// MustFieldByName find `aorm.Field` with field name or db name
func (this *Instance) MustFieldByName(name string) (field *Field) {
	field, _ = this.FieldByName(name)
	return
}

func (this *Instance) ID() ID {
	return this.Struct.GetID(this.Value)
}

func (this *ModelStruct) CreateFieldByName(value interface{}, fieldName string) (field *Field, ok bool) {
	var (
		indirectScopeValue = indirect(reflect.ValueOf(value))
		structField        = this.FieldsByName[fieldName]
		fieldValue         = indirectScopeValue.FieldByIndex(structField.StructIndex)
	)
	if structField == nil {
		return
	}
	return &Field{StructField: structField, Field: fieldValue, IsBlank: IsBlank(fieldValue)}, true
}

func (this *ModelStruct) MustCreateFieldByName(value interface{}, fieldName string) (field *Field) {
	field, ok := this.CreateFieldByName(value, fieldName)
	if !ok {
		panic(fmt.Errorf("field %q does not exists", fieldName))
	}
	return field
}

// FieldByName find `aorm.StructField` with field name or db name
func (this *ModelStruct) FieldByName(name string) (field *StructField, ok bool) {
	if field, ok = this.FieldsByName[name]; ok {
		return
	}
	var (
		dbName           = ToDBName(name)
		mostMatchedField *StructField
	)
	for _, field := range this.Fields {
		if field.DBName == name {
			return field, true
		}
		if field.DBName == dbName {
			mostMatchedField = field
		}
	}
	return mostMatchedField, mostMatchedField != nil
}

func (this *ModelStruct) newFields() (fields *Instance) {
	return this.InstanceOf(this.Value)
}

func (this *ModelStruct) InstanceOf(value interface{}, fieldsNames ...string) (fields *Instance) {
	var (
		indirectScopeValue = indirect(reflect.ValueOf(value))
		isStruct           = indirectScopeValue.Kind() == reflect.Struct
		byName             = make(map[string]*Field)
		field              *Field
	)
	fields = &Instance{Struct: this, Value: value, ReflectValue: indirectScopeValue, FieldsMap: byName}

	var sfields []*StructField
	if len(fieldsNames) == 0 {
		sfields = this.Fields
	} else {
		for _, name := range fieldsNames {
			sfields = append(sfields, this.FieldsByName[name])
		}
	}

	if isStruct {
		for _, structField := range sfields {
			fieldValue := indirectScopeValue.FieldByIndex(structField.StructIndex)
			field = &Field{StructField: structField, Field: fieldValue, IsBlank: IsBlank(fieldValue)}
			fields.Fields = append(fields.Fields, field)
			if _, ok := byName[field.Name]; !ok {
				byName[field.Name] = field
			}
			if structField.IsPrimaryKey {
				fields.Primary = append(fields.Primary, field)
			}
		}
	} else {
		for _, structField := range sfields {
			field = &Field{StructField: structField, IsBlank: true}

			fields.Fields = append(fields.Fields, field)
			if _, ok := byName[field.Name]; !ok {
				fields.FieldsMap[field.Name] = field
			}
		}
	}
	return
}

func (this *ModelStruct) FirstFieldValue(value interface{}, names ...string) (field *Field) {
	var (
		indirectScopeValue = indirect(reflect.ValueOf(value))
	)
	for _, name := range names {
		if structField, ok := this.FieldsByName[name]; ok {
			fieldValue := indirectScopeValue.FieldByIndex(structField.StructIndex)
			return &Field{StructField: structField, Field: fieldValue, IsBlank: IsBlank(fieldValue)}
		}
	}
	return
}
