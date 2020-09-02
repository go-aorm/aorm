package aorm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

func (scope *Scope) queryToString(clause interface{}) (result Query) {
	switch value := clause.(type) {
	case string:
		result.Query = value
	case []string:
		result.Query = strings.Join(value, ", ")
	case []interface{}:
		var columns []string
		for _, arg := range value {
			c := scope.queryToString(arg)
			if c.Query != "" {
				columns = append(columns, c.Query)
				result.AddArgs(c.Args...)
			}
		}
		result.Query = strings.Join(columns, ", ")
	case WhereClauser:
		return value.WhereClause(scope)
	case *Alias:
		result.Query = value.Expr + " AS " + value.Name
	}
	return
}

func (scope *Scope) buildSelectQuery(clause *Clause) (result *Query) {
	result = &Query{}
	var columns []string
	c := scope.queryToString(clause.Query)
	result.AddArgs(c.Args...)
	columns = append(columns, c.Query)
	for _, arg := range clause.Args {
		switch reflect.ValueOf(arg).Kind() {
		case reflect.Slice:
			values := reflect.ValueOf(arg)
			for i := 0; i < values.Len(); i++ {
				result.AddArgs(values.Index(i).Interface())
				columns = append(columns, "?")
			}
		default:
			result.AddArgs(arg)
			columns = append(columns, "?")
		}
	}
	result.Query = strings.Join(columns, ", ")
	return
}

func (scope *Scope) whereSQL() (sql string) {
	var (
		quotedTableName                                = scope.QuotedTableName()
		deletedAtField, hasDeletedAtField              = scope.Struct().FieldsByName["DeletedAt"]
		primaryConditions, andConditions, orConditions []string
	)

	if !scope.Search.Unscoped && hasDeletedAtField {
		sql := fmt.Sprintf("%v.%v IS NULL", quotedTableName, scope.Quote(deletedAtField.DBName))
		primaryConditions = append(primaryConditions, sql)
	}

	rt := indirectType(reflect.TypeOf(scope.Value))
	switch rt.Kind() {
	case reflect.Struct:
		if s := scope.Struct(); s.HasID() {
			if s.PrimaryField().StructIndex == nil {
				if id, ok := scope.InstanceGet("aorm:id"); ok {
					scope.Search.whereConditions = append(scope.Search.whereConditions, nil)
					copy(scope.Search.whereConditions[1:], scope.Search.whereConditions)
					scope.Search.whereConditions[0] = &Clause{Query: id.(ID)}
				}
			} else {
				if id := scope.Struct().GetID(scope.Value); !id.IsZero() {
					for _, field := range scope.PrimaryFields() {
						sql := fmt.Sprintf("%v.%v = %v", quotedTableName, scope.Quote(field.DBName), scope.AddToVars(field.Field.Interface()))
						primaryConditions = append(primaryConditions, sql)
					}
				}
			}
		}
	}

	for _, clause := range scope.Search.whereConditions {
		if sql, err := clause.BuildCondition(scope, true).Build(scope); err != nil {
			scope.Err(err)
		} else if sql != "" {
			andConditions = append(andConditions, sql)
		}
	}

	for _, clause := range scope.Search.orConditions {
		if sql, err := clause.BuildCondition(scope, true).Build(scope); err != nil {
			scope.Err(err)
		} else if sql != "" {
			orConditions = append(orConditions, sql)
		}
	}

	for _, clause := range scope.Search.notConditions {
		if sql, err := clause.BuildCondition(scope, false).Build(scope); err != nil {
			scope.Err(err)
		} else if sql != "" {
			andConditions = append(andConditions, sql)
		}
	}

	orSQL := strings.Join(orConditions, " OR ")
	combinedSQL := strings.Join(andConditions, " AND ")
	if len(combinedSQL) > 0 {
		if len(orSQL) > 0 {
			combinedSQL = combinedSQL + " OR " + orSQL
		}
	} else {
		combinedSQL = orSQL
	}

	if len(primaryConditions) > 0 {
		sql = "WHERE " + strings.Join(primaryConditions, " AND ")
		if len(combinedSQL) > 0 {
			sql = sql + " AND (" + combinedSQL + ")"
		}
	} else if len(combinedSQL) > 0 {
		sql = "WHERE " + combinedSQL
	}
	return
}

func (scope *Scope) selectSQL() (sql string) {
	if scope.Search.selects == nil {
		var tbName string
		//if len(scope.Search.joinConditions) > 0 || len(scope.Search.inlinePreload) {
		tbName = scope.QuotedTableName() + "."
		//}
		var columns []string
		for _, f := range scope.Struct().Fields {
			if f.IsNormal || f.Selector != nil {
				if f.IsPrimaryKey && scope.Search.ignorePrimaryFields {
					continue
				}
				if q, err := f.Select(scope, tbName).Build(scope); err != nil {
					scope.Err(errors.Wrapf(err, "select sql for field %s", f))
					return ""
				} else if q != "" {
					columns = append(columns, q)
				}
			}
		}
		sql = strings.Join(columns, ", ")
	} else {
		if q, ok := scope.Search.selects["query"]; ok {
			var (
				clause Clause
				err    error
			)
			clause.Query = q
			if args, ok := scope.Search.selects["args"]; ok {
				clause.Args = args.([]interface{})
			}
			if sql, err = scope.buildSelectQuery(&clause).Build(scope); err != nil {
				scope.Err(err)
				return ""
			}
		}
	}
	if !scope.fixedColumns {
		if scope.Search.extraSelects != nil {
			for _, es := range scope.Search.extraSelects.Items {
				if sql_, err := scope.buildSelectQuery(&es.Clause).Build(scope); err != nil {
					scope.Err(err)
					return ""
				} else if sql_ != "" {
					sql += "," + sql_
				}
			}
		}
		if scope.Search.extraSelectsFields != nil {
			for _, es := range scope.Search.extraSelectsFields.Items {
				if sql_, err := scope.buildSelectQuery(&es.Clause).Build(scope); err != nil {
					scope.Err(err)
					return ""
				} else if sql_ != "" {
					sql += ", " + sql_
				}
			}
		}
	}
	return
}

func (scope *Scope) orderSQL() string {
	if len(scope.Search.orders) == 0 || scope.Search.ignoreOrderQuery {
		return ""
	}

	var orders []string
	for _, order := range scope.Search.orders {
		if clause := scope.Clause(order); clause != "" {
			orders = append(orders, clause)
		}
	}
	return " ORDER BY " + strings.Join(orders, ",")
}

func (scope *Scope) limitAndOffsetSQL() string {
	return scope.Dialect().LimitAndOffsetSQL(scope.Search.limit, scope.Search.offset)
}

func (scope *Scope) groupSQL() string {
	if len(scope.Search.group) == 0 {
		return ""
	}
	return " GROUP BY " + scope.Search.group
}

func (scope *Scope) havingSQL() string {
	if len(scope.Search.havingConditions) == 0 {
		return ""
	}

	var andConditions []string
	for _, clause := range scope.Search.havingConditions {
		if q := clause.BuildCondition(scope, true); q.Query != "" {
			if c, err := q.Build(scope); err != nil {
				scope.Err(err)
				return ""
			} else {
				andConditions = append(andConditions, c)
			}
		}
	}

	combinedSQL := strings.Join(andConditions, " AND ")
	if len(combinedSQL) == 0 {
		return ""
	}

	return " HAVING " + combinedSQL
}

func (scope *Scope) joinsSQL() string {
	var joinConditions []string
	for _, clause := range scope.Search.joinConditions {
		if q := clause.BuildCondition(scope, true); q.Query != "" {
			if sql, err := q.Build(scope); err != nil {
				scope.Err(err)
				return ""
			} else {
				joinConditions = append(joinConditions, strings.TrimSuffix(strings.TrimPrefix(sql, "("), ")"))
			}
		}
	}

	return strings.Join(joinConditions, " ") + " "
}

func (scope *Scope) prepareQuerySQL() {
	if scope.Search.raw {
		scope.Raw(scope.CombinedConditionSql())
	} else {
		scope.Raw(fmt.Sprintf("SELECT %v FROM %v %v", scope.selectSQL(), scope.fromSql(), scope.CombinedConditionSql()))
	}
	return
}
