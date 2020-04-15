package aorm

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

type Clause struct {
	Query interface{}
	Args  []interface{}
}

func (clause *Clause) BuildCondition(scope *Scope, include bool) (result Query) {
	var (
		quotedTableName  = scope.QuotedTableName()
		quotedPrimaryKey = scope.Quote(scope.PrimaryKeyDbName())
		equalSQL         = "="
		inSQL            = "IN"
	)

	// If building not conditions
	if !include {
		equalSQL = "<>"
		inSQL = "NOT IN"
	}

	switch value := clause.Query.(type) {
	case *Query:
		return *value
	case Query:
		return value
	case []WhereClauser:
		var (
			s    = make([]string, len(value))
			args []interface{}
		)
		for _, clauser := range value {
			c := clauser.WhereClause(scope)
			s = append(s, c.Query)
			result.Args = append(result.Args, args...)
		}
		result.Query = "(" + strings.Join(s, " AND ") + ")"
		result.AddArgs(clause.Args...)
		return
	case WhereClauser:
		result = value.WhereClause(scope)
		result.AddArgs(clause.Args...)
		return
	/*case *WithInlineQuery:
	defer func() {
		clause.Query = value
	}()
	clause.Query = value.Merge(scope)
	return scope.buildCondition(clause, include)*/
	case sql.NullInt64:
		result.Query = fmt.Sprintf("(%v.%v %s %v)", quotedTableName, quotedPrimaryKey, equalSQL, value.Int64)
		return
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		result.Query = fmt.Sprintf("(%v.%v %s %v)", quotedTableName, quotedPrimaryKey, equalSQL, value)
		return
	case []int, []int8, []int16, []int32, []int64, []uint, []uint8, []uint16, []uint32, []uint64, []string, []interface{}:
		if !include && reflect.ValueOf(value).Len() == 0 {
			return
		}
		result.Query = fmt.Sprintf("(%v.%v %s (?))", quotedTableName, quotedPrimaryKey, inSQL)
		result.AddArgs(value)
		return
	case string:
		if value == "" {
			return
		}
		if relField, ok := scope.Struct().FieldsByName[value]; ok && relField.Relationship != nil {
			panic("not implemented")
			return
		}
		if isNumberRegexp.MatchString(value) {
			result.Query = fmt.Sprintf("(%v.%v %s ?)", quotedTableName, quotedPrimaryKey, equalSQL)
			result.AddArgs(value)
			return
		}

		if !include {
			if comparisonRegexp.MatchString(value) {
				result.Query = fmt.Sprintf("NOT (%v)", value)
			} else {
				result.Query = fmt.Sprintf("(%v.%v NOT IN (?))", quotedTableName, scope.Quote(value))
			}
		} else {
			result.Query = fmt.Sprintf("(%v)", value)
		}
		return *result.AddArgs(clause.Args...)
	case map[string]interface{}:
		var sqls []string
		for key, value := range value {
			if value != nil {
				sqls = append(sqls, fmt.Sprintf("(%v.%v %s ?)", quotedTableName, scope.Quote(key), equalSQL))
				result.AddArgs(value)
				continue
			}

			if !include {
				sqls = append(sqls, fmt.Sprintf("(%v.%v IS NOT NULL)", quotedTableName, scope.Quote(key)))
			} else {
				sqls = append(sqls, fmt.Sprintf("(%v.%v IS NULL)", quotedTableName, scope.Quote(key)))
			}
		}
		result.Query = strings.Join(sqls, " AND ")
		return
	case interface{}:
		var sqls []string
		newScope := scope.New(value)

		if len(newScope.Instance().Fields) == 0 {
			scope.Err(fmt.Errorf("invalid query condition: %v", value))
			return
		}

		for _, field := range newScope.Instance().Fields {
			if !field.IsIgnored && !field.IsBlank && field.Relationship == nil {
				sqls = append(sqls, fmt.Sprintf("(%v.%v %s ?)", quotedTableName, scope.Quote(field.DBName), equalSQL))
				result.AddArgs(field.Field.Interface())
			}
		}
		result.Query = strings.Join(sqls, " AND ")
		return
	default:
		scope.Err(fmt.Errorf("invalid query condition: %v", value))
		return
	}
}
