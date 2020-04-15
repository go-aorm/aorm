package aorm

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"

	"github.com/jinzhu/inflection"
)

// DefaultTableNameHandler default table name handler
var DefaultTableNameHandler = func(ctx context.Context, singular bool, modelStruct *ModelStruct) (tableName string) {
	if singular {
		return modelStruct.SingularTableName
	}
	return modelStruct.PluralTableName
}

// ModelStruct model definition
type ModelStruct struct {
	storage                        *ModelStructStorage
	TableNameResolver              func(ctx context.Context, singular bool) (tableName string)
	Value                          interface{}
	PrimaryFields                  []*StructField
	Fields                         []*StructField
	RelatedFields                  []*StructField
	ReadOnlyFields                 []*StructField
	DynamicFieldsByName            map[string]*StructField
	Type                           reflect.Type
	PluralTableName                string
	SingularTableName              string
	FieldsByName                   map[string]*StructField
	IgnoredFieldsCount             int
	BeforeRelatedCallbacks         []func(fromScope *Scope, toScope *Scope, db *DB, fromField *Field) *DB
	virtualFields                  map[string]*VirtualField
	virtualFieldsByIndex           []*VirtualField
	virtualFieldsAutoInlinePreload []string
	softDelete                     bool
	Indexes                        IndexMap
	UniqueIndexes                  IndexMap
}

func (this *ModelStruct) Fqn() string {
	return this.Type.PkgPath() + "." + this.Type.Name()
}

func (this *ModelStruct) BeforeRelatedCallback(cb ...func(fromScope *Scope, toScope *Scope, db *DB, fromField *Field) *DB) {
	this.BeforeRelatedCallbacks = append(this.BeforeRelatedCallbacks, cb...)
}

// tableName get model's table name
func (this *ModelStruct) TableName(ctx context.Context, singular bool) string {
	if ctx == nil {
		ctx = context.Background()
	}
	if this.TableNameResolver != nil {
		if tableName := this.TableNameResolver(ctx, singular); tableName != "" {
			return tableName
		}
	}
	if this.PluralTableName == "" && this.Type != nil {
		// Set default table name
		if tabler, ok := this.Value.(TableNamePlurabler); ok {
			this.PluralTableName = tabler.TableName(false)
			this.SingularTableName = tabler.TableName(true)
		} else if tabler, ok := this.Value.(TableNamer); ok {
			this.PluralTableName = tabler.TableName()
			this.SingularTableName = this.PluralTableName
		} else {
			this.SingularTableName = ToDBName(this.Type.Name())
			this.PluralTableName = inflection.Plural(this.SingularTableName)
		}
	}
	return DefaultTableNameHandler(ctx, singular, this)
}

func (this *ModelStruct) IsSoftDelete() bool {
	return this.softDelete
}

func (this *ModelStruct) SetVirtualField(fieldName string, value interface{}) *VirtualField {
	if this.virtualFields != nil {
		if _, ok := this.virtualFields[fieldName]; ok {
			panic(fmt.Errorf("Duplicate virtual field %q", fieldName))
		}
	} else {
		this.virtualFields = map[string]*VirtualField{}
	}
	var ms, err = this.storage.GetOrNew(value)
	if err != nil {
		panic(err)
	}
	vf := &VirtualField{
		ModelStruct: ms,
		FieldName:   fieldName,
		StructIndex: len(this.virtualFieldsByIndex),
		Value:       value,
		Options:     map[interface{}]interface{}{},
	}
	this.virtualFieldsByIndex = append(this.virtualFieldsByIndex, vf)
	this.virtualFields[fieldName] = vf
	return vf
}

func (this *ModelStruct) GetVirtualField(fieldName string) *VirtualField {
	if this.virtualFields == nil {
		return nil
	}
	return this.virtualFields[fieldName]
}

// GetNonIgnoredStructFields get non ignored model's field structs
func (this *ModelStruct) NonIgnoredStructFields() []*StructField {
	fields := make([]*StructField, len(this.Fields)-this.IgnoredFieldsCount)
	var i int
	for _, field := range this.Fields {
		if !field.IsIgnored {
			fields[i] = field
			i++
		}
	}
	return fields
}

// NonRelatedStructFields get non ignored model's field structs
func (this *ModelStruct) NonRelatedStructFields() []*StructField {
	fields := make([]*StructField, len(this.Fields)-this.IgnoredFieldsCount)
	var i int
	for _, field := range this.Fields {
		if !field.IsIgnored && field.Relationship == nil && field.TagSettings["FOREIGNKEY"] == "" {
			fields[i] = field
			i++
		}
	}
	return fields[0:i]
}

// GetNonIgnoredStructFields get non ignored model's field structs
func (this *ModelStruct) NormalStructFields() (fields []*StructField) {
	for _, field := range this.Fields {
		if !field.IsIgnored && !field.IsForeignKey {
			fields = append(fields, field)
		}
	}
	return fields
}

// AutoInlinePreload set default auto inline preload virtual field names
func (this *ModelStruct) AutoInlinePreload(virtualFieldName ...string) {
	this.virtualFieldsAutoInlinePreload = append(this.virtualFieldsAutoInlinePreload, virtualFieldName...)
}

// FieldDiscovery discovery field from name or path
func (this *ModelStruct) FieldDiscovery(pth string) (field *StructField, virtualField *VirtualField) {
	currentModelStruct := this
	parts := strings.Split(pth, ".")

	for _, fieldName := range parts[0 : len(parts)-1] {
		if f, ok := currentModelStruct.FieldsByName[fieldName]; ok {
			typ := f.Struct.Type
			switch typ.Kind() {
			case reflect.Slice, reflect.Ptr:
				typ = typ.Elem()
			}
			currentModelStruct = modelStructsMap.Get(typ)
		} else {
			if vfield := currentModelStruct.GetVirtualField(fieldName); vfield == nil {
				return
			} else {
				currentModelStruct = vfield.ModelStruct
			}
		}
	}

	fieldName := parts[len(parts)-1]
	var ok bool

	if field, ok = currentModelStruct.FieldsByName[fieldName]; !ok {
		virtualField = currentModelStruct.GetVirtualField(fieldName)
	}

	return
}

func (this *ModelStruct) SetIdFromString(record interface{}, idstr string) (err error) {
	var id ID
	if id, err = this.ParseIDString(idstr); err != nil {
		return
	}
	id.SetTo(record)
	return
}

func (this *ModelStruct) GetID(record interface{}) ID {
	if record == nil {
		return nil
	}
	var (
		rv reflect.Value
		ok bool
	)
	if rv, ok = record.(reflect.Value); !ok {
		rv = reflect.ValueOf(record)
	}
	rv = indirect(rv)

	switch len(this.PrimaryFields) {
	case 0:
		return nil
	default:
		var values []IDValuer
		for _, f := range this.PrimaryFields {
			rv := rv.FieldByIndex(f.StructIndex)
			if valuer, err := f.IDOf(rv.Interface()); err != nil {
				panic(errors.Wrapf(err, "field %s#%q", this.Fqn(), f.Name))
			} else {
				values = append(values, valuer)
			}
		}
		return NewId(this.PrimaryFields, values)
	}
}

func (this *ModelStruct) PrimaryFieldsInstance(value interface{}) (fields []*Field) {
	if value == nil {
		return
	}
	var (
		indirectScopeValue = indirect(reflect.ValueOf(value))
	)

	if indirectScopeValue.Kind() != reflect.Struct {
		return
	}
	for _, structField := range this.PrimaryFields {
		fieldValue := indirectScopeValue.FieldByIndex(structField.StructIndex)
		fields = append(fields, &Field{StructField: structField, Field: fieldValue, IsBlank: IsBlank(fieldValue)})
	}
	return
}

// RealTableName get real table name
func (this *ModelStruct) RealTableName(ctx context.Context, singular bool) (name string) {
	if tabler, ok := this.Value.(TableNamer); ok {
		return tabler.TableName()
	}
	return this.TableName(ctx, singular)
}

// PrimaryField return main primary field, if defined more that one primary fields, will return the one having column name `id` or the first one
func (this *ModelStruct) PrimaryField() *StructField {
	if len(this.PrimaryFields) > 0 {
		for _, f := range this.PrimaryFields {
			if f.DBName == "id" {
				return f
			}
		}
		return this.PrimaryFields[0]
	}
	return nil
}

// HasID returns if has main primary field
func (this *ModelStruct) HasID() bool {
	return len(this.PrimaryFields) > 0
}

// FieldByPath return field byte path
func (this *ModelStruct) FieldByPath(pth string) (field *StructField) {
	fields := strings.Split(pth, ".")
	self := this
	var err error
	for i, field := range fields[:len(fields)-1] {
		if self, err = this.storage.GetOrNew(self.FieldsByName[field].Struct.Type); err != nil {
			panic(errors.Wrapf(err, "path %q", strings.Join(fields[0:i], ".")))
		}
	}

	return self.FieldsByName[fields[len(fields)-1]]
}

// UniqueIdexesNamesMap return unique idexes by index name map
func (this *ModelStruct) UniqueIdexesNamesMap(namer KeyNamer, tableName string) map[string]*StructIndex {
	var result = make(map[string]*StructIndex)
	for _, ix := range this.UniqueIndexes {
		result[ix.BuildName(namer, tableName)] = ix
	}
	return result
}

func (this *ModelStruct) DefaultID() ID {
	var values = make([]IDValuer, len(this.PrimaryFields), len(this.PrimaryFields))
	for i, f := range this.PrimaryFields {
		var err error
		values[i], err = f.DefaultID()
		if err != nil {
			panic(err)
		}
	}
	return NewId(this.PrimaryFields, values)
}
