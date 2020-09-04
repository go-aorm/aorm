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
	ScopeCallbacks
	TypeCallbacks

	Name  string
	Value interface{}

	Parent,
	parentTemp *ModelStruct
	HasManyChild bool

	storage                        *ModelStructStorage
	ParentField                    *StructField
	TableNameResolver              func(ctx context.Context, singular bool) (tableName string)
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
	Children                       []*ModelStruct
	ChildrenByName                 map[string]*ModelStruct
	HasManyChildren                []*ModelStruct
	HasManyChildrenByName          map[string]*ModelStruct
	InlinePreloadFields            []string
	ForeignKeys                    []*ForeignKey
	ParentForeignKey               *ForeignKey
	Tags                           TagSetting
}

func (this *ModelStruct) Clone() *ModelStruct {
	var clone = *this
	clone.DynamicFieldsByName = map[string]*StructField{}
	clone.FieldsByName = map[string]*StructField{}
	clone.virtualFields = map[string]*VirtualField{}
	clone.ChildrenByName = map[string]*ModelStruct{}
	clone.HasManyChildrenByName = map[string]*ModelStruct{}

	for k, v := range this.DynamicFieldsByName {
		clone.DynamicFieldsByName[k] = v
	}
	for k, v := range this.FieldsByName {
		clone.FieldsByName[k] = v
	}
	for k, v := range this.virtualFields {
		clone.virtualFields[k] = v
	}
	for k, v := range this.ChildrenByName {
		clone.ChildrenByName[k] = v
	}
	for k, v := range this.HasManyChildrenByName {
		clone.HasManyChildrenByName[k] = v
	}

	clone.ScopeCallbacks.Registrator = *this.ScopeCallbacks.Registrator.Clone()
	clone.TypeCallbacks.TypeRegistrator = *this.TypeCallbacks.TypeRegistrator.Clone()
	return &clone
}

func (this *ModelStruct) PkgPath() string {
	for this.Type.PkgPath() == "" {
		this = this.Parent
	}
	return this.Type.PkgPath()
}

func (this *ModelStruct) Fqn() string {
	var name = this.Type.Name()
	if this.Name != name {
		name += "@" + this.Name
	}
	return this.PkgName() + "." + name
}

func (this *ModelStruct) BeforeRelatedCallback(cb ...func(fromScope *Scope, toScope *Scope, db *DB, fromField *Field) *DB) {
	this.BeforeRelatedCallbacks = append(this.BeforeRelatedCallbacks, cb...)
}

// tableName get model's table name
func (this *ModelStruct) PkgName() (pkg string) {
	orignalPath := this.PkgPath()
	if pkg = PkgNamer.Get(orignalPath); pkg != "" {
		return
	}
	return orignalPath
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

	if singular {
		if this.SingularTableName != "" {
			return this.SingularTableName
		}
	} else if this.PluralTableName != "" {
		return this.PluralTableName
	}

	if this.Type != nil {
		if this.Parent == nil {
			prefix := TableNamePrefixOf(this.PkgPath())
			if tname := this.Tags["TABLE_NAME"]; tname != "" {
				this.Tags["TABLE"] = tname
				delete(this.Tags, "TABLE_NAME")
			}
			if tname := this.Tags.GetTags("TABLE"); tname != nil {
				if this.SingularTableName = tname.GetStringAlias("SINGULAR", "S"); this.SingularTableName != "" && this.SingularTableName[0] == '.' {
					this.SingularTableName = prefix + "_" + this.SingularTableName[1:]
				}
				if this.PluralTableName = tname.GetStringAlias("PLURAL", "P"); this.PluralTableName != "" && this.PluralTableName[0] == '.' {
					this.PluralTableName = prefix + "_" + this.PluralTableName[1:]
				}

				if singular {
					if this.SingularTableName != "" {
						return this.SingularTableName
					}
				} else if this.PluralTableName != "" {
					return this.PluralTableName
				}
			} else if tname := this.Tags.GetString("TABLE"); tname != "" {
				if tname[0] == '.' {
					tname = prefix + "_" + tname[1:]
				}
				this.SingularTableName = tname
				this.PluralTableName = inflection.Plural(tname)
			}

			// Set default table name
			if tabler, ok := this.Value.(TableNamePlurabler); ok {
				if this.SingularTableName == "" {
					if this.SingularTableName = tabler.TableName(true); this.SingularTableName[0] == '.' {
						this.SingularTableName = TableNameOfPrefix(prefix, this.SingularTableName)[0]
					}
				}
				if this.PluralTableName == "" {
					if this.PluralTableName = tabler.TableName(false); this.PluralTableName[0] == '.' {
						this.PluralTableName = TableNameOfPrefix(prefix, this.PluralTableName)[0]
					}
				}
			} else if tabler, ok := this.Value.(TableNamer); ok {
				if this.PluralTableName == "" {
					this.PluralTableName = tabler.TableName()
					if this.PluralTableName[0] == '.' {
						this.PluralTableName = TableNameOfPrefix(prefix, this.PluralTableName)[0]
					}
				}
				if this.SingularTableName == "" {
					this.SingularTableName = this.PluralTableName
				}
			} else {
				var name = this.Name
				if name == "" {
					name = this.Type.Name()
				}
				if this.SingularTableName == "" {
					this.SingularTableName = TableNameOfPrefix(prefix, ToDBName(name))[0]
				}
				if this.PluralTableName == "" {
					this.PluralTableName = inflection.Plural(this.SingularTableName)
				}
			}
		} else {
			singular, plural := ChildName(this.ParentField)
			dbName := ToDBName(singular)
			this.SingularTableName = this.Parent.TableName(ctx, false) + "__" + dbName
			this.PluralTableName = this.Parent.TableName(ctx, false) + "__" + ToDBName(plural)
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
		Model:       ms,
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

type FieldQuery struct {
	FieldName       string
	Struct          *StructField
	Virtual         *VirtualField
	InlineQueryName string
}

// FieldPathQueryOf returns field path query of field name or path
func (this *ModelStruct) FieldPathQueryOf(fieldName string) (iq *FieldPathQuery) {
	iq = &FieldPathQuery{}
	var rFieldName string
	if field, ok := this.FieldsByName[fieldName]; ok {
		if field.Relationship != nil && (field.Relationship.Kind == "belongs_to" || field.Relationship.Kind == "has_one") {
			rFieldName = field.Relationship.ForeignFieldNames[0]
			iq.Struct = field.BaseModel.FieldsByName[rFieldName]
			iq.SetQuery("{}." + iq.Struct.DBName)
			return
		}
	}
	var ok bool
	if iq.Struct, iq.Virtual, ok = this.FieldDiscovery(fieldName); !ok {
		return nil
	}
	if iq.Struct != nil {
		if iq.Struct.Relationship != nil {
			typ := iq.Struct.Struct.Type
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
			iq.Struct = iq.Struct.Relationship.AssociationModel.PrimaryField()
			rFieldName = fieldName + "." + iq.Struct.Name
		} else if iq.Struct.BaseModel.Parent != nil {
			typ := iq.Struct.Struct.Type
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
			// iq.Struct = iq.Struct.BaseModel.ParentField.Relationship.AssociationModel.PrimaryField()
			rFieldName = fieldName
		}
		parts := strings.Split(rFieldName, ".")
		iq.SetQuery("{" + strings.Join(parts[0:len(parts)-1], ".") + "}." + iq.Struct.DBName)
	}
	return
}

// FieldDiscovery discovery field from name or path
func (this *ModelStruct) FieldDiscovery(pth string) (field *StructField, virtualField *VirtualField, ok bool) {
	currentModelStruct := this
	parts := strings.Split(pth, ".")

	for _, fieldName := range parts[0 : len(parts)-1] {
		if f, ok2 := currentModelStruct.FieldsByName[fieldName]; ok2 {
			typ := f.Struct.Type
			switch typ.Kind() {
			case reflect.Slice, reflect.Ptr:
				typ = typ.Elem()
			}
			if f.Model != nil {
				currentModelStruct = f.Model
			} else {
				currentModelStruct = modelStructsMap.Get(typ)
			}
		} else {
			if vfield := currentModelStruct.GetVirtualField(fieldName); vfield == nil {
				return
			} else {
				currentModelStruct = vfield.Model
			}
		}

		if currentModelStruct == nil {
			return nil, nil, false
		}
	}

	fieldName := parts[len(parts)-1]
	if field, ok = currentModelStruct.FieldsByName[fieldName]; !ok {
		virtualField = currentModelStruct.GetVirtualField(fieldName)
	}

	ok = field != nil || virtualField != nil
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
		if len(values) == 0 {
			return nil
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
