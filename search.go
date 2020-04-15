package aorm

import (
	"fmt"
	"strings"
)

type search struct {
	db                     *DB
	whereConditions        []*Clause
	orConditions           []*Clause
	notConditions          []*Clause
	havingConditions       []*Clause
	joinConditions         []*Clause
	initAttrs              []interface{}
	assignAttrs            []interface{}
	selects                map[string]interface{}
	omits                  []string
	orders                 []interface{}
	preload                []searchPreload
	inlinePreload          []searchPreload
	offset                 interface{}
	limit                  interface{}
	group                  string
	from                   string
	tableName              string
	tableAlias             string
	raw                    bool
	Unscoped               bool
	ignoreOrderQuery       bool
	extraSelects           *extraSelects
	extraSelectsFields     *extraSelectsFields
	defaultColumnValue     func(scope *Scope, record interface{}, column string) interface{}
	columnsScannerCallback func(scope *Scope, record interface{}, columns []string, values []interface{})
}

type searchPreload struct {
	schema  string
	options *InlinePreloadOptions
}

func (s *search) clone() *search {
	clone := *s
	return &clone
}

func (s search) resetConditions() *search {
	s.whereConditions = nil
	s.orConditions = nil
	s.notConditions = nil
	return &s
}

func (s *search) Where(query interface{}, values ...interface{}) *search {
	s.whereConditions = append(s.whereConditions, &Clause{query, values})
	return s
}

func (s *search) Not(query interface{}, values ...interface{}) *search {
	s.notConditions = append(s.notConditions, &Clause{query, values})
	return s
}

func (s *search) Or(query interface{}, values ...interface{}) *search {
	s.orConditions = append(s.orConditions, &Clause{query, values})
	return s
}

func (s *search) Attrs(attrs ...interface{}) *search {
	s.initAttrs = append(s.initAttrs, toSearchableMap(attrs...))
	return s
}

func (s *search) Assign(attrs ...interface{}) *search {
	s.assignAttrs = append(s.assignAttrs, toSearchableMap(attrs...))
	return s
}

func (s *search) Order(value interface{}, reorder ...bool) *search {
	if len(reorder) > 0 && reorder[0] {
		s.orders = []interface{}{}
	}

	if value != nil && value != "" {
		if sl, ok := value.([]interface{}); ok {
			for _, value := range sl {
				s.Order(value)
			}
		} else {
			s.orders = append(s.orders, value)
		}
	}
	return s
}

func (s *search) Select(query interface{}, args ...interface{}) *search {
	s.selects = map[string]interface{}{"query": query, "args": args}
	return s
}

func (s *search) ExtraSelect(key string, values []interface{}, query interface{}, args ...interface{}) *search {
	if s.extraSelects == nil {
		s.extraSelects = &extraSelects{}
	}
	s.extraSelects.Add(key, values, query, args)
	return s
}

func (s *search) ExtraSelectCallback(f ...func(recorde interface{}, data map[string]*ExtraResult)) *search {
	if s.extraSelects == nil {
		s.extraSelects = &extraSelects{}
	}
	s.extraSelects.Callback(f...)
	return s
}

func (s *search) ExtraSelectFields(key string, value interface{}, fields []*StructField, callback func(scope *Scope, record interface{}), query interface{}, args ...interface{}) *search {
	if s.extraSelectsFields == nil {
		s.extraSelectsFields = &extraSelectsFields{}
	}
	s.extraSelectsFields.Add(key, value, fields, callback, query, args)
	return s
}

// ExtraSelectFields specify extra fields that you want to retrieve from database when querying
func (s *search) ExtraSelectFieldsSetter(key string, setter ExtraSelectFieldsSetter, structFields []*StructField, query interface{}, args ...interface{}) *DB {
	return s.ExtraSelectFields(key, setter, structFields, nil, query, args...).db
}

func (s *search) Omit(columns ...string) *search {
	s.omits = columns
	return s
}

func (s *search) Limit(limit interface{}) *search {
	s.limit = limit
	return s
}

func (s *search) Offset(offset interface{}) *search {
	s.offset = offset
	return s
}

func (s *search) Group(query string) *search {
	s.group = s.getInterfaceAsSQL(query)
	return s
}

func (s *search) Having(query interface{}, values ...interface{}) *search {
	if val, ok := query.(*Query); ok {
		s.havingConditions = append(s.havingConditions, &Clause{val.Query, val.Args})
	} else {
		s.havingConditions = append(s.havingConditions, &Clause{query, values})
	}
	return s
}

func (s *search) Joins(query string, values ...interface{}) *search {
	s.joinConditions = append(s.joinConditions, &Clause{query, values})
	return s
}

func (s *search) Preload(schema string, options ...*InlinePreloadOptions) *search {
	var opt *InlinePreloadOptions
	if len(options) > 0 {
		opt = options[0]
	} else {
		opt = &InlinePreloadOptions{}
	}
	var preloads []searchPreload
	for _, preload := range s.preload {
		if preload.schema != schema {
			preloads = append(preloads, preload)
		}
	}
	preloads = append(preloads, searchPreload{schema, opt})
	s.preload = preloads
	return s
}

func (s *search) InlinePreload(schema string, options ...*InlinePreloadOptions) *search {
	var opt *InlinePreloadOptions
	for _, opt = range options {
	}
	if opt == nil {
		opt = &InlinePreloadOptions{}
	}
	var preloads []searchPreload
	for _, preload := range s.inlinePreload {
		if preload.schema != schema {
			preloads = append(preloads, preload)
		}
	}
	preloads = append(preloads, searchPreload{schema, opt})
	s.inlinePreload = preloads
	return s
}

func (s *search) Raw(b bool) *search {
	s.raw = b
	return s
}

func (s *search) unscoped() *search {
	s.Unscoped = true
	return s
}

func (s *search) Table(name string) *search {
	// ... as ALIAS
	if index := strings.LastIndex(name, " "); index != -1 {
		s.tableAlias = name[index:]
		// remove " as ALIAS"
		name = name[0 : index-4]
		// from (select ...) as ALIAS
		if name[0] == '(' {
			// without parentesis
			name = name[1 : len(name)-1]
			s.from = name
			return s
		}
	}
	s.tableName = name
	return s
}

func (s *search) From(from string) *search {
	s.from = from
	return s
}

func (s *search) getInterfaceAsSQL(value interface{}) (str string) {
	switch value.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		str = fmt.Sprintf("%v", value)
	default:
		s.db.AddError(ErrInvalidSQL)
	}

	if str == "-1" {
		return ""
	}
	return
}
