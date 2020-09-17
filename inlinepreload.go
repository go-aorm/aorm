package aorm

import (
	"fmt"
	"reflect"
	"strings"

	tag_scanner "github.com/unapu-go/tag-scanner"
)

type InlinePreloadInfo struct {
	RootScope, ParentScope, Scope *Scope
	Preloader                     *InlinePreloader
	Conditions                    *Conditions
}

type InlinePreloadOptions struct {
	Conditions
	RelatedConditions []interface{}
	Select            []interface{}
	Join              JoinType
	Prepare           func(builder *InlinePreloadBuilder)
}

type InlinePreloadBuilder struct {
	*Conditions
	*InlinePreloadInfo
}

type InlinePreloader struct {
	RootScope,
	Scope,
	ParentScope *Scope
	DB             *DB
	ID             string
	Field          *StructField
	VirtualField   *VirtualField
	Index          [][]int
	RelationFields []*StructField
	StructFields   []*StructField
	Query          string
}

func (p *InlinePreloader) Fields(fields ...interface{}) {
	for _, f := range fields {
		switch ft := f.(type) {
		case *StructField:
			p.StructFields = append(p.StructFields, ft)
		case []*StructField:
			p.StructFields = append(p.StructFields, ft...)
		case string:
			if field, ok := p.Scope.Struct().FieldsByName[ft]; ok {
				p.StructFields = append(p.StructFields, field)
			} else if ft == "*" {
				for _, field := range p.Scope.GetNonRelatedStructFields() {
					p.StructFields = append(p.StructFields, field)
				}
			} else {
				p.RootScope.Err(fmt.Errorf("Struct field %q does not exists.", ft))
			}
		case []string:
			for _, fieldName := range ft {
				if fieldName == "*" {
					for _, field := range p.Scope.GetNonRelatedStructFields() {
						p.StructFields = append(p.StructFields, field)
					}
					continue
				}
				if field, ok := p.Scope.Struct().FieldsByName[fieldName]; ok {
					p.StructFields = append(p.StructFields, field)
				} else {
					p.RootScope.Err(fmt.Errorf("Struct field %q does not exists.", fieldName))
				}
			}
		}
	}

	var newFields []*StructField

	for _, f := range p.StructFields {
		if f.IsReadOnly || (p.Scope.Search.ignorePrimaryFields && f.IsPrimaryKey) {
			continue
		}
		if f.Relationship == nil {
			newFields = append(newFields, f)
		} else {
			p.RelationFields = append(p.RelationFields, f)
		}
	}

	p.StructFields = newFields

	if p.Scope.Search.ignorePrimaryFields {
		return
	}

KeyFields:
	for _, kf := range p.Scope.Struct().PrimaryFields {
		for _, f := range p.StructFields {
			if f.Name == kf.Name {
				continue KeyFields
			}
		}
		p.StructFields = append([]*StructField{kf}, p.StructFields...)
	}
}

func (p *InlinePreloader) GetFields() []*StructField {
	if len(p.StructFields) == 0 {
		if len(p.Scope.modelStruct.Children) > 0 {
			var names []string
			for _, model := range p.Scope.modelStruct.Children {
				names = append(names, model.ParentField.Name)
			}
			p.Fields(names)
		}
		if irf, ok := p.Scope.Value.(InlinePreloadFields); ok {
			p.Fields(irf.GetAormInlinePreloadFields())
			if len(p.StructFields) != 0 || len(p.RelationFields) != 0 {
				return p.StructFields
			}
		} else if irf, ok := p.Scope.Value.(InlinePreloadFieldsWithScope); ok {
			p.Fields(irf.GetAormInlinePreloadFields(p.ParentScope))
			if len(p.StructFields) != 0 || len(p.RelationFields) != 0 {
				return p.StructFields
			}
		}
		if len(p.Scope.modelStruct.InlinePreloadFields) > 0 {
			p.Fields(p.Scope.modelStruct.InlinePreloadFields)
		}
		if p.Field != nil {
			if preload := p.Field.TagSettings["PRELOAD"]; preload != "" {
				if p.Field.TagSettings.Scanner().IsTags(preload) {
					p.Fields(tag_scanner.Flags(p.Field.TagSettings.Scanner(), preload))
				} else {
					p.Fields(strings.Split(preload, ","))
				}
				if len(p.StructFields) != 0 {
					return p.StructFields
				}
			}
		}

		if p.Scope.Search.ignorePrimaryFields {
			for _, f := range p.Scope.GetNonRelatedStructFields() {
				if !f.IsPrimaryKey {
					p.StructFields = append(p.StructFields, f)
				}
			}
		} else {
			for _, f := range p.Scope.GetNonRelatedStructFields() {
				if !f.IsReadOnly {
					p.StructFields = append(p.StructFields, f)
				}
			}
		}
	}
	return p.StructFields
}

func (p *InlinePreloader) GetQuery() string {
	if p.Query == "" {
		fields := p.GetFields()
		columns := make([]string, len(fields))
		for i, f := range fields {
			columns[i] = fmt.Sprintf("%v.%v", p.ID, p.Scope.Quote(f.DBName))
		}
		p.Query = strings.Join(columns, ", ")
	}
	return p.Query
}

func (p *InlinePreloader) Select() {
	fields := p.GetFields()
	if !p.RootScope.counter {
		p.RootScope.Search.ExtraSelectFieldsSetter(p.ID, p.Scan, fields, p.GetQuery())
	}
}

func (p *InlinePreloader) Scan(result interface{}, values []interface{}, set func(model *ModelStruct, result interface{}, low, hight int) interface{}) {
	if p.Scope.Search.ignorePrimaryFields {
		for _, v := range values {
			if v.(*ValueScanner).NotNil {
				goto scan
			}
		}
		// all values is nil
		return
	} else if values[0].(*ValueScanner).IsNil() {
		// first field is PK, if is nil, no have values
		return
	}
scan:
	var model *ModelStruct
	if p.Field != nil {
		model = p.Field.Model
	} else {
		model = p.VirtualField.Model
	}
	field := reflect.Indirect(reflect.ValueOf(result))
	ms := p.RootScope.Struct()
	for _, pth := range p.Index {
		if len(pth) == 1 && pth[0] < 0 {
			i := (pth[0] * -1) - 1
			vf := ms.virtualFieldsByIndex[i]
			if mvf, ok := field.Addr().Interface().(ModelWithVirtualFields); ok {
				v, ok := mvf.GetVirtualField(vf.FieldName)
				if ok {
					field = reflect.Indirect(reflect.ValueOf(v))
				} else {
					rv := reflect.New(vf.Model.Type)
					v = rv.Interface()
					vf.Set(mvf, v)
					field = rv
				}
			} else if vf.Getter != nil {
				if v, ok := vf.Getter(vf, field.Addr().Interface()); ok {
					field = reflect.Indirect(reflect.ValueOf(v))
				} else {
					rv := reflect.New(vf.Model.Type)
					v = rv.Interface()
					vf.Set(field.Addr().Interface(), v)
					field = rv
				}
			}
		} else {
			field = field.FieldByIndex(pth)
			if field.Kind() == reflect.Ptr && isNil(field) {
				field.Set(reflect.New(field.Type().Elem()))
			}
		}
		field = reflect.Indirect(field)
	}
	set(model, field, 0, 0)
	if cb, ok := result.(AfterInlinePreloadScanner); ok {
		cb.AormAfterInlinePreloadScan(p, result, field.Addr().Interface())
	}
}

type InlinePreloads struct {
	Count uint
	// map of field path -> alias_name
	DBNames map[string]string
}

func (c *InlinePreloads) Next(fieldPath ...string) string {
	v := c.Count
	c.Count++
	dbName := fmt.Sprintf("aorm_prl_%d", v)
	if c.DBNames == nil {
		c.DBNames = map[string]string{}
	}
	c.DBNames[strings.Join(fieldPath, ".")] = dbName
	return dbName
}

func (c *InlinePreloads) GetDBName(path string) (dbName string, ok bool) {
	if c.DBNames == nil {
		return
	}
	dbName, ok = c.DBNames[path]
	return
}

func InlinePreloadFieldsKeyOf(value interface{}) string {
	typ := indirectType(reflect.TypeOf(value))
	return "inline_preload_fields:" + typ.PkgPath() + "." + typ.Name()
}
