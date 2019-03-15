package aorm

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
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
	rootScope, scope *Scope
	DB               *DB
	ID               string
	Field            *StructField
	VirtualField     *VirtualField
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
				if fieldName == "*" {
					for _, field := range p.scope.GetNonRelatedStructFields() {
						p.StructFields = append(p.StructFields, field)
					}
					continue
				}
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
			if len(p.StructFields) != 0 || len(p.RelationFields) != 0 {
				return p.StructFields
			}
		}
		if p.Field != nil {
			if preload := p.Field.TagSettings["PRELOAD"]; preload != "" {
				p.Fields(strings.Split(preload, ","))
				if len(p.StructFields) != 0 {
					return p.StructFields
				}
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
	field := p.GetFields()
	if !p.rootScope.counter {
		p.rootScope.Search.ExtraSelectFieldsSetter(p.ID, p.Scan, field, p.GetQuery())
	}
}

func (p *InlinePreloader) Scan(result interface{}, values []interface{}, set func(result interface{}, low, hight int) interface{}) {
	if !values[0].(*ValueScanner).IsNil() {
		field := reflect.Indirect(reflect.ValueOf(result))
		ms := p.rootScope.GetModelStruct()
		for _, pth := range p.Index {
			if len(pth) == 1 && pth[0] < 0 {
				i := (pth[0] * -1) - 1
				vf := ms.virtualFieldsByIndex[i]
				if mvf, ok := field.Addr().Interface().(ModelWithVirtualFields); ok {
					v, ok := mvf.GetVirtualField(vf.FieldName)
					if ok {
						field = reflect.Indirect(reflect.ValueOf(v))
					} else {
						rv := reflect.New(vf.ModelStruct.ModelType)
						v = rv.Interface()
						vf.Set(mvf, v)
						field = rv
					}
				} else if vf.Getter != nil {
					if v, ok := vf.Getter(vf, field.Addr().Interface()); ok {
						field = reflect.Indirect(reflect.ValueOf(v))
					} else {
						rv := reflect.New(vf.ModelStruct.ModelType)
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
		set(field, 0, 0)
		if cb, ok := result.(AfterInlinePreloadScanner); ok {
			cb.AormAfterInlinePreloadScan(p, result, field.Addr().Interface())
		}
	}
}

type InlinePreloadFields interface {
	GetGormInlinePreloadFields() []string
}

type InlinePreloads struct {
	Count uint
	// map of field path -> alias_name
	DBNames map[string]string
}

func (c *InlinePreloads) Next(fieldPath ...string) string {
	v := c.Count
	c.Count++
	dbName := fmt.Sprintf("gorm_prl_%d", v)
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

type WithInlineQuery struct {
	query string
	paths []string
}

func IQ(query string) *WithInlineQuery {
	paths := PathsFromQuery(query)
	return &WithInlineQuery{query, paths}
}

func (iq *WithInlineQuery) Query() string {
	return iq.query
}

func (iq *WithInlineQuery) Paths() []string {
	return iq.paths
}

func (iq *WithInlineQuery) Merge(scope *Scope) string {
	query := iq.query
	tbName := scope.TableName()
	for _, p := range iq.paths {
		if p == "" {
			query = strings.Replace(query, "{}", tbName, -1)
		} else {
			query = strings.Replace(query, "{"+p+"}", scope.inlinePreloads.DBNames[p], -1)
		}
	}
	return query
}

func (iq *WithInlineQuery) String() string {
	return iq.query
}

type InlineQueries []*WithInlineQuery

func (iq InlineQueries) Join(sep ...string) (result *WithInlineQuery) {
	var (
		ok      bool
		queries []string

		s  = ", "
		pm = map[string]bool{}
	)
	if len(sep) > 0 {
		s = sep[0]
	}
	result = &WithInlineQuery{}

	for _, iq := range iq {
		for _, p := range iq.paths {
			if ok, _ = pm[p]; !ok {
				pm[p] = true
				result.paths = append(result.paths, p)
			}
		}
		queries = append(queries, iq.query)
	}

	result.query = strings.Join(queries, s)
	sort.Strings(result.paths)
	return
}

var fieldPathRegex, _ = regexp.Compile(`\{(|\w+(\.\w+)*)\}`)

func PathsFromQuery(query string) (paths []string) {
	var (
		p  string
		ok bool
		pm = map[string]bool{}
	)

	for _, match := range fieldPathRegex.FindAllStringSubmatch(query, -1) {
		p = match[1]
		if ok, _ = pm[p]; !ok {
			pm[p] = true
			paths = append(paths, p)
		}
	}
	return
}

type AfterInlinePreloadScanner interface {
	AormAfterInlinePreloadScan(ip *InlinePreloader, recorde, value interface{})
}

func InlinePreloadFieldsKeyOf(value interface{}) string {
	typ := indirectType(reflect.TypeOf(value))
	return "inline_preload_fields:" + typ.PkgPath() + "." + typ.Name()
}
