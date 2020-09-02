package aorm

import (
	"errors"
	"fmt"
	"reflect"
)

// Define callbacks for querying
func init() {
	DefaultCallback.Query().Register("aorm:start", startQueryStart)
	DefaultCallback.Query().Register("aorm:inline_preload", inlinePreloadCallback)
	DefaultCallback.Query().Register("aorm:query", queryCallback)
	DefaultCallback.Query().Register("aorm:preload", preloadCallback)
	DefaultCallback.Query().Register("aorm:after_query", afterQueryCallback)
}

// startQueryStart starts query callbacks
func startQueryStart(scope *Scope) {
	scope.Operation = OpQuery
}

// queryCallback used to query data from database
func queryCallback(scope *Scope) {
	if _, skip := scope.InstanceGet("aorm:skip_query_callback"); skip {
		return
	}

	scope.ExecTime = NowFunc()
	defer scope.trace(scope.ExecTime)

	var (
		resultType      reflect.Type
		results, sender = scope.ResultSender()
	)

	if !scope.Search.raw {
		if orderBy, ok := scope.Get("aorm:order_by_primary_key"); ok {
			if primaryField := scope.Struct().PrimaryField(); primaryField != nil {
				scope.Search.Order(fmt.Sprintf("%v.%v %v", scope.QuotedTableName(), scope.Quote(primaryField.DBName), orderBy))
			}
		}
	}

	if value, ok := scope.Get("aorm:query_destination"); ok {
		results = reflect.ValueOf(value)
		sender = SenderOf(results)
	}

	if sender != nil {
		if results.Elem().Kind() == reflect.Chan {
			results = results.Elem()
		}
	} else {
		results = results.Elem()
	}

	resultType, _, _ = StructTypeOf(results.Type())

	if resultType.Kind() != reflect.Struct {
		scope.Err(errors.New("unsupported destination, should be slice or struct"))
		return
	}

	scope.prepareQuerySQL()

	if scope.HasError() {
		return
	}

	scope.db.RowsAffected = 0
	if str, ok := scope.Get("aorm:query_option"); ok {
		scope.Query.Query += addExtraSpaceIfExist(fmt.Sprint(str))
	}

	if scope.checkDryRun() {
		return
	}

	rows := scope.log(LOG_READ).runQueryRows()
	if scope.HasError() {
		return
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	for rows.Next() {
		scope.db.RowsAffected++

		elem := results
		if sender != nil {
			elem = reflect.New(resultType).Elem()
		}

		result := scope.New(elem.Addr().Interface())
		scope.scan(rows, columns, result.Instance().Fields, elem.Addr().Interface())

		if !scope.HasError() {
			if acv, ok := elem.Interface().(interface {
				AfterScan(*Scope)
			}); ok {
				acv.AfterScan(scope)
			} else if acv, ok := elem.Interface().(interface {
				AfterScan(*DB)
			}); ok {
				acv.AfterScan(scope.DB())
			} else if acv, ok := elem.Addr().Interface().(interface {
				AfterScan(*Scope)
			}); ok {
				acv.AfterScan(scope)
			} else if acv, ok := elem.Addr().Interface().(interface {
				AfterScan(*DB)
			}); ok {
				acv.AfterScan(scope.DB())
			}
		}

		if sender != nil {
			sender(elem)
		}
	}

	if err := rows.Err(); err != nil {
		scope.Err(err)
	} else if scope.db.RowsAffected == 0 && sender == nil {
		scope.Err(ErrRecordNotFound)
	}
}

// afterQueryCallback will invoke `AfterFind` method after querying
func afterQueryCallback(scope *Scope) {
	if !scope.HasError() {
		scope.CallMethod("AfterFind")
	}
}
