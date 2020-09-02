package aorm

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type FieldPathQuery struct {
	Struct  *StructField
	Virtual *VirtualField
	query   string
	paths   []string
	args    []interface{}
}

func NewFieldPathQuery(field *StructField, virtual *VirtualField, query string, args ...interface{}) *FieldPathQuery {
	return &FieldPathQuery{Struct: field, Virtual: virtual, query: query, paths: PathsFromQuery(query), args: args}
}

func IQ(query string, args ...interface{}) *FieldPathQuery {
	return NewFieldPathQuery(nil, nil, query, args...)
}

func (this *FieldPathQuery) SetQuery(query string) {
	this.query = query
	this.paths = PathsFromQuery(query)
}

func (this *FieldPathQuery) Query() string {
	return this.query
}

func (this *FieldPathQuery) Paths() []string {
	return this.paths
}

func (this *FieldPathQuery) Prefix(v string) *FieldPathQuery {
	this.query = v + this.query
	return this
}

func (this *FieldPathQuery) Sufix(v string) *FieldPathQuery {
	this.query += v
	return this
}

func (this *FieldPathQuery) WhereClause(scope *Scope) Query {
	query := this.query
	tbName := scope.QuotedTableName()
	for _, p := range this.paths {
		if dbName, ok := scope.inlinePreloads.DBNames[p]; ok {
			query = strings.ReplaceAll(query, "{"+p+"}", dbName)
		} else if p == "" {
			query = strings.ReplaceAll(query, "{}", tbName)
		} else if f := scope.Struct().FieldsByName[p]; f != nil {
			if f.Relationship != nil {
				dbName := scope.ScopeOfField(f.Name).QuotedTableName()
				query = strings.ReplaceAll(query, "{"+p+"}", dbName)
			}
		} else {
			panic(fmt.Errorf("inline preload db name of %q does not exists", p))
		}
	}
	return Query{query, this.args}
}

func (this *FieldPathQuery) String() string {
	return this.query
}

type InlineQueries []*FieldPathQuery

func (iq InlineQueries) Join(sep ...string) (result *FieldPathQuery) {
	var (
		ok      bool
		queries []string

		s  = ", "
		pm = map[string]bool{}
	)
	if len(sep) > 0 {
		s = sep[0]
	}
	result = &FieldPathQuery{}

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
