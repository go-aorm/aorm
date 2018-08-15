package gorm

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type InlinePreloader struct {
	rootScope, scope *Scope
	DB               *DB
	ID               string
	Field            *StructField
	Index            [][]int
	RelationFields   []*StructField
	StructFields     []*StructField
	Query            string
}

func (p *InlinePreloader) Fields(fields ...interface{}) {
	for _, f := range fields {
		switch ft := f.(type) {
		case *StructField:
			p.StructFields = append(p.StructFields, ft)
		case []*StructField:
			p.StructFields = append(p.StructFields, ft...)
		case string:
			if field, ok := p.scope.GetModelStruct().StructFieldsByName[ft]; ok {
				p.StructFields = append(p.StructFields, field)
			} else {
				p.rootScope.Err(fmt.Errorf("Struct field %q does not exists.", ft))
			}
		case []string:
			for _, fieldName := range ft {
				if field, ok := p.scope.GetModelStruct().StructFieldsByName[fieldName]; ok {
					p.StructFields = append(p.StructFields, field)
				} else {
					p.rootScope.Err(fmt.Errorf("Struct field %q does not exists.", fieldName))
				}
			}
		}
	}

	var newFields []*StructField

	for _, f := range p.StructFields {
		if f.Relationship == nil {
			newFields = append(newFields, f)
		} else {
			p.RelationFields = append(p.RelationFields, f)
		}
	}

	p.StructFields = newFields

KeyFields:
	for _, kf := range p.scope.GetModelStruct().PrimaryFields {
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
		if irf, ok := p.scope.Value.(InlinePreloadFields); ok {
			p.Fields(irf.GetGormInlinePreloadFields())
			if len(p.StructFields) != 0 {
				return p.StructFields
			}
		}
		if preload := p.Field.TagSettings["PRELOAD"]; preload != "" {
			p.Fields(strings.Split(preload, ","))
			if len(p.StructFields) != 0 {
				return p.StructFields
			}
		}
		p.StructFields = p.scope.GetNonRelatedStructFields()
	}
	return p.StructFields
}

func (p *InlinePreloader) GetQuery() string {
	if p.Query == "" {
		fields := p.GetFields()
		columns := make([]string, len(fields))
		for i, f := range fields {
			columns[i] = fmt.Sprintf("%v.%v", p.ID, p.scope.Quote(f.DBName))
		}
		p.Query = strings.Join(columns, ", ")
	}
	return p.Query
}

func (p *InlinePreloader) Apply() {
	p.rootScope.Search.ExtraSelectFieldsSetter(p.ID, p.Scan, p.GetFields(), p.GetQuery())
}

func (p *InlinePreloader) Scan(result interface{}, values []interface{}, set func(result interface{}, low, hight int) interface{}) {
	if !values[0].(*ValueScanner).IsNil() {
		field := reflect.ValueOf(result).Elem()
		for _, pth := range p.Index {
			field = field.FieldByIndex(pth)
		}
		set(field, 0, 0)
	}
}

type InlinePreloadFields interface {
	GetGormInlinePreloadFields() []string
}

type InlinePreloadCounter struct {
	Count uint
}

func (c *InlinePreloadCounter) Next() uint {
	v := c.Count
	c.Count++
	return v
}

func (c *InlinePreloadCounter) NextS() string {
	return strconv.Itoa(int(c.Next()))
}
