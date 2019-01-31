package aorm

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
)

type Conditions struct {
	whereConditions []map[string]interface{}
	orConditions    []map[string]interface{}
	notConditions   []map[string]interface{}
	Args            []interface{}
}

func (c *Conditions) Prepare(cb func(c *Conditions, query interface{}, args []interface{}, replace func(query interface{}, args ...interface{}))) {
	iter := func(c *Conditions, l *[]map[string]interface{}) {
		for i, cond := range *l {
			cb(c, cond["query"], cond["args"].([]interface{}), func(query interface{}, args ...interface{}) {
				if query == nil {
					(*l)[i] = nil
				} else {
					(*l)[i]["query"], (*l)[i]["args"] = query, args
				}
			})
		}
	}

	var (
		where, or, not []map[string]interface{}
		args           []interface{}
	)
	cur := c
	for cur.hasConditions() {
		newC := &Conditions{}
		iter(newC, &cur.whereConditions)
		iter(newC, &cur.orConditions)
		iter(newC, &cur.notConditions)
		where, or, not = append(where, cur.whereConditions...), append(or, cur.orConditions...), append(not, cur.notConditions...)
		args = append(args, cur.Args...)
		cur = newC
	}
	c.whereConditions, c.orConditions, c.notConditions = where, or, not
}

// AddToVars add value as sql's vars, used to prevent SQL injection
func (c *Conditions) AddToVars(scope *Scope, value interface{}) string {
	_, skipBindVar := scope.InstanceGet("skip_bindvar")

	if expr, ok := value.(*expr); ok {
		exp := expr.expr
		for _, arg := range expr.args {
			if skipBindVar {
				c.AddToVars(scope, arg)
			} else {
				exp = strings.Replace(exp, "?", c.AddToVars(scope, arg), 1)
			}
		}
		return exp
	}

	if assigner := scope.db.GetAssigner(reflect.TypeOf(value)); assigner != nil {
		value = assigner.Valuer(scope.db.Dialect(), value)
	}

	c.Args = append(c.Args, value)

	if skipBindVar {
		return "?"
	}
	return scope.Dialect().BindVar(len(c.Args))
}

func (c *Conditions) Where(query interface{}, values ...interface{}) *Conditions {
	c.whereConditions = append(c.whereConditions, map[string]interface{}{"query": query, "args": values})
	return c
}

func (c *Conditions) Not(query interface{}, values ...interface{}) *Conditions {
	c.notConditions = append(c.notConditions, map[string]interface{}{"query": query, "args": values})
	return c
}

func (c *Conditions) Or(query interface{}, values ...interface{}) *Conditions {
	c.orConditions = append(c.orConditions, map[string]interface{}{"query": query, "args": values})
	return c
}

func (c *Conditions) hasConditions() bool {
	return len(c.whereConditions) > 0 ||
		len(c.orConditions) > 0 ||
		len(c.notConditions) > 0
}

func (c *Conditions) MergeTo(db *DB) *DB {
	db = db.clone()
	db.search.whereConditions = append(db.search.whereConditions, c.whereConditions...)
	db.search.orConditions = append(db.search.orConditions, c.orConditions...)
	db.search.notConditions = append(db.search.notConditions, c.notConditions...)
	return db
}

func (c *Conditions) buildCondition(scope *Scope, clause map[string]interface{}, include bool) (str string) {
	if clause == nil {
		return
	}
	var (
		quotedTableName  = scope.QuotedTableName()
		quotedPrimaryKey = scope.Quote(scope.PrimaryKey())
		equalSQL         = "="
		inSQL            = "IN"
	)

	// If building not conditions
	if !include {
		equalSQL = "<>"
		inSQL = "NOT IN"
	}

	switch value := clause["query"].(type) {
	case *WithInlineQuery:
		defer func() {
			clause["query"] = value
		}()
		clause["query"] = value.Merge(scope)
		return c.buildCondition(scope, clause, include)
	case sql.NullInt64:
		return fmt.Sprintf("(%v.%v %s %v)", quotedTableName, quotedPrimaryKey, equalSQL, value.Int64)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("(%v.%v %s %v)", quotedTableName, quotedPrimaryKey, equalSQL, value)
	case KeyInterface:
		var (
			pkfields = scope.PrimaryFields()
			values = value.Values()
		)
		var s = make([]string, len(pkfields))
		for i, f := range pkfields {
			s[i] = fmt.Sprintf("%v.%v %s %v", quotedTableName, scope.Quote(f.DBName), c.AddToVars(scope, values[i]))
		}
		return "(" + strings.Join(s, " AND ") + ")"
	case []int, []int8, []int16, []int32, []int64, []uint, []uint8, []uint16, []uint32, []uint64, []string, []interface{}:
		if !include && reflect.ValueOf(value).Len() == 0 {
			return
		}
		str = fmt.Sprintf("(%v.%v %s (?))", quotedTableName, quotedPrimaryKey, inSQL)
		clause["args"] = []interface{}{value}
	case string:
		if isNumberRegexp.MatchString(value) {
			return fmt.Sprintf("(%v.%v %s %v)", quotedTableName, quotedPrimaryKey, equalSQL, c.AddToVars(scope, value))
		}

		if value != "" {
			if !include {
				if comparisonRegexp.MatchString(value) {
					str = fmt.Sprintf("NOT (%v)", value)
				} else {
					str = fmt.Sprintf("(%v.%v NOT IN (?))", quotedTableName, scope.Quote(value))
				}
			} else {
				str = fmt.Sprintf("(%v)", value)
			}
		}
	case map[string]interface{}:
		var sqls []string
		for key, value := range value {
			if value != nil {
				sqls = append(sqls, fmt.Sprintf("(%v.%v %s %v)", quotedTableName, scope.Quote(key), equalSQL, c.AddToVars(scope, value)))
			} else {
				if !include {
					sqls = append(sqls, fmt.Sprintf("(%v.%v IS NOT NULL)", quotedTableName, scope.Quote(key)))
				} else {
					sqls = append(sqls, fmt.Sprintf("(%v.%v IS NULL)", quotedTableName, scope.Quote(key)))
				}
			}
		}
		return strings.Join(sqls, " AND ")
	case interface{}:
		var sqls []string
		newScope := scope.New(value)

		if len(newScope.Fields()) == 0 {
			scope.Err(fmt.Errorf("invalid query condition: %v", value))
			return
		}

		for _, field := range newScope.Fields() {
			if !field.IsIgnored && !field.IsBlank {
				sqls = append(sqls, fmt.Sprintf("(%v.%v %s %v)", quotedTableName, scope.Quote(field.DBName), equalSQL, c.AddToVars(scope, field.Field.Interface())))
			}
		}
		return strings.Join(sqls, " AND ")
	default:
		scope.Err(fmt.Errorf("invalid query condition: %v", value))
		return
	}

	replacements := []string{}
	args := clause["args"].([]interface{})
	for _, arg := range args {
		var err error
		switch reflect.ValueOf(arg).Kind() {
		case reflect.Slice: // For where("id in (?)", []int64{1,2})
			if scanner, ok := interface{}(arg).(driver.Valuer); ok {
				arg, err = scanner.Value()
				replacements = append(replacements, c.AddToVars(scope, arg))
			} else if b, ok := arg.([]byte); ok {
				replacements = append(replacements, c.AddToVars(scope, b))
			} else if as, ok := arg.([][]interface{}); ok {
				var tempMarks []string
				for _, a := range as {
					var arrayMarks []string
					for _, v := range a {
						arrayMarks = append(arrayMarks, c.AddToVars(scope, v))
					}

					if len(arrayMarks) > 0 {
						tempMarks = append(tempMarks, fmt.Sprintf("(%v)", strings.Join(arrayMarks, ",")))
					}
				}

				if len(tempMarks) > 0 {
					replacements = append(replacements, strings.Join(tempMarks, ","))
				}
			} else if values := reflect.ValueOf(arg); values.Len() > 0 {
				var tempMarks []string
				for i := 0; i < values.Len(); i++ {
					tempMarks = append(tempMarks, c.AddToVars(scope, values.Index(i).Interface()))
				}
				replacements = append(replacements, strings.Join(tempMarks, ","))
			} else {
				replacements = append(replacements, c.AddToVars(scope, Expr("NULL")))
			}
		default:
			if valuer, ok := interface{}(arg).(driver.Valuer); ok {
				arg, err = valuer.Value()
			}

			replacements = append(replacements, c.AddToVars(scope, arg))
		}

		if err != nil {
			scope.Err(err)
		}
	}

	buff := bytes.NewBuffer([]byte{})
	i := 0
	for _, s := range str {
		if s == '?' && len(replacements) > i {
			buff.WriteString(replacements[i])
			i++
		} else {
			buff.WriteRune(s)
		}
	}

	str = buff.String()

	return
}

func (c *Conditions) whereSQL(scope *Scope) (sql string) {
	var (
		primaryConditions, andConditions, orConditions []string
	)

	for _, clause := range c.whereConditions {
		if sql := c.buildCondition(scope, clause, true); sql != "" {
			andConditions = append(andConditions, sql)
		}
	}

	for _, clause := range c.orConditions {
		if sql := c.buildCondition(scope, clause, true); sql != "" {
			orConditions = append(orConditions, sql)
		}
	}

	for _, clause := range c.notConditions {
		if sql := c.buildCondition(scope, clause, false); sql != "" {
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
		sql = strings.Join(primaryConditions, " AND ")
		if len(combinedSQL) > 0 {
			sql = sql + " AND (" + combinedSQL + ")"
		}
	} else if len(combinedSQL) > 0 {
		sql = combinedSQL
	}
	return
}
