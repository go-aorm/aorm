package aorm

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type WithInlineQuery struct {
	query string
	paths []string
	args  []interface{}
}

func IQ(query string, args ...interface{}) *WithInlineQuery {
	paths := PathsFromQuery(query)
	return &WithInlineQuery{query, paths, args}
}

func (iq *WithInlineQuery) Query() string {
	return iq.query
}

func (iq *WithInlineQuery) Paths() []string {
	return iq.paths
}

func (iq *WithInlineQuery) WhereClause(scope *Scope) Query {
	query := iq.query
	tbName := scope.TableName()
	for _, p := range iq.paths {
		if p == "" {
			query = strings.Replace(query, "{}", tbName, -1)
		} else if f := scope.Struct().FieldsByName[p]; f != nil {
			if f.Relationship != nil {
				dbName := scope.ScopeOfField(f.Name).TableName()
				query = strings.Replace(query, "{"+p+"}", dbName, -1)
			}
		} else if dbName, ok := scope.inlinePreloads.DBNames[p]; ok {
			query = strings.Replace(query, "{"+p+"}", dbName, -1)
		} else {
			panic(fmt.Errorf("inline preload db name of %q does not exists", p))
		}
	}
	return Query{query, iq.args}
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
