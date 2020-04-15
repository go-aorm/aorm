package aorm

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	CbGenId = ""
)

// Define callbacks for creating
func init() {
	DefaultCallback.Create().Register("aorm:start", startCreateCallback)
	DefaultCallback.Create().Register("aorm:set_id", SetIdCallback)
	DefaultCallback.Create().Register("aorm:begin_transaction", beginTransactionCallback)
	DefaultCallback.Create().Register("aorm:before_create", beforeCreateCallback)
	DefaultCallback.Create().Register("aorm:save_before_associations", saveBeforeAssociationsCallback)
	DefaultCallback.Create().Register("aorm:update_time_stamp", updateTimeStampForCreateCallback)
	DefaultCallback.Create().Register("aorm:audited", auditedForCreateCallback)
	DefaultCallback.Create().Register("aorm:create", createCallback)
	DefaultCallback.Create().Register("aorm:force_reload_after_create", forceReloadAfterCreateCallback)
	DefaultCallback.Create().Register("aorm:save_after_associations", saveAfterAssociationsCallback)
	DefaultCallback.Create().Register("aorm:after_create", afterCreateCallback)
	DefaultCallback.Create().Register("aorm:commit_or_rollback_transaction", commitOrRollbackTransactionCallback)
}

// startCreateCallback starts creating callbacks
func startCreateCallback(scope *Scope) {
	scope.Operation = OpCreate
}

// beforeCreateCallback will invoke `BeforeSave`, `BeforeCreate` method before creating
func beforeCreateCallback(scope *Scope) {
	if !scope.HasError() {
		scope.CallMethod("BeforeSave")
	}
	if !scope.HasError() {
		scope.CallMethod("BeforeCreate")
	}
}

// updateTimeStampForCreateCallback will set `CreatedAt`, `UpdatedAt` when creating
func updateTimeStampForCreateCallback(scope *Scope) {
	if !scope.HasError() {
		now := NowFunc()

		if createdAtField, ok := scope.FieldByName("CreatedAt"); ok {
			if createdAtField.IsBlank {
				createdAtField.Set(now)
			}
		}

		if updatedAtField, ok := scope.FieldByName("UpdatedAt"); ok {
			if updatedAtField.IsBlank {
				updatedAtField.Set(now)
			}
		}
	}
}

// auditedForCreateCallback will set `CreatedByID` when updating
func auditedForCreateCallback(scope *Scope) {
	if _, ok := scope.Get("aorm:created_by_column"); !ok {
		if user := scope.GetCurrentUserID(); user != nil {
			scope.SetColumn("CreatedByID", user)
		}
	}
}

// createCallback the callback used to insert data into database
func createCallback(scope *Scope) {
	scope.ExecTime = NowFunc()
	defer scope.trace(scope.ExecTime)

	var (
		columns, placeholders        []string
		blankColumnsWithDefaultValue []string
	)

	for _, field := range scope.Instance().Fields {
		if scope.changeableField(field) {
			if field.IsNormal {
				if field.IsBlank && field.HasDefaultValue {
					blankColumnsWithDefaultValue = append(blankColumnsWithDefaultValue, scope.Quote(field.DBName))
					scope.InstanceSet("aorm:blank_columns_with_default_value", blankColumnsWithDefaultValue)

					// default expression
					if field.TagSettings["DEFAULT"] == "DEFAULT" {
						fv := reflect.ValueOf(scope.Value).MethodByName("Aorm" + field.Name + "DefaultCreationQuery")
						if fv.IsValid() {
							columns = append(columns, scope.Quote(field.DBName))
							result := fv.Call([]reflect.Value{reflect.ValueOf(scope)})
							expression, args := result[0].Interface().(string), result[1].Interface().([]interface{})
							placeholders = append(placeholders, scope.AddToVars(Expr("("+expression+")", args...)))
						}
					}
				} else if !field.IsPrimaryKey || !field.IsBlank {
					columns = append(columns, scope.Quote(field.DBName))
					placeholders = append(placeholders, scope.AddToVars(field.Field.Interface()))
				}
			} else if field.Relationship != nil && field.Relationship.Kind == "belongs_to" {
				for _, foreignKey := range field.Relationship.ForeignDBNames {
					if foreignField, ok := scope.FieldByName(foreignKey); ok && !scope.changeableField(foreignField) {
						columns = append(columns, scope.Quote(foreignField.DBName))
						placeholders = append(placeholders, scope.AddToVars(foreignField.Field.Interface()))
					}
				}
			}
		}
	}

	var (
		returningColumn = "*"
		quotedTableName = scope.QuotedTableName()
		primaryField    = scope.PrimaryField()
		extraOption     string
	)

	if str, ok := scope.Get("aorm:insert_option"); ok {
		extraOption = fmt.Sprint(str)
	}

	if primaryField != nil {
		returningColumn = scope.Quote(primaryField.DBName)
	}

	lastInsertIDReturningSuffix := scope.Dialect().LastInsertIDReturningSuffix(quotedTableName, returningColumn)

	if len(columns) == 0 {
		scope.Raw(fmt.Sprintf(
			"INSERT INTO %v %v%v%v",
			quotedTableName,
			scope.Dialect().DefaultValueStr(),
			addExtraSpaceIfExist(extraOption),
			addExtraSpaceIfExist(lastInsertIDReturningSuffix),
		))
	} else {
		scope.Raw(fmt.Sprintf(
			"INSERT INTO %v (%v) VALUES (%v)%v%v",
			scope.QuotedTableName(),
			strings.Join(columns, ","),
			strings.Join(placeholders, ","),
			addExtraSpaceIfExist(extraOption),
			addExtraSpaceIfExist(lastInsertIDReturningSuffix),
		))
	}

	if scope.checkDryRun() {
		return
	}

	// execute create sql
	if lastInsertIDReturningSuffix == "" || primaryField == nil {
		scope.log(LOG_CREATE)
		if result := scope.ExecResult(); !scope.HasError() {
			// set primary value to primary field
			if primaryField != nil && primaryField.IsBlank {
				if primaryValue, err := result.LastInsertId(); scope.Err(err) == nil {
					scope.Err(primaryField.Set(primaryValue))
				}
			}
		}
	} else {
		if primaryField.Field.CanAddr() {
			scope.log(LOG_CREATE)
			if err := scope.SQLDB().QueryRow(scope.Query.Query, scope.Query.Args...).Scan(primaryField.Field.Addr().Interface()); scope.Err(err) == nil {
				primaryField.IsBlank = false
				scope.db.RowsAffected = 1
			}
		} else {
			scope.Err(ErrUnaddressable)
		}
	}
}

// forceReloadAfterCreateCallback will reload columns that having default value, and set it back to current object
func forceReloadAfterCreateCallback(scope *Scope) {
	if blankColumnsWithDefaultValue, ok := scope.InstanceGet("aorm:blank_columns_with_default_value"); ok {
		db := scope.DB().New().Table(scope.TableName()).Select(blankColumnsWithDefaultValue.([]string))
		for _, field := range scope.Instance().Primary {
			if !field.IsBlank {
				db = db.Where(fmt.Sprintf("%v = ?", field.DBName), field.Field.Interface())
			}
		}
		db.Scan(scope.Value)
	}
}

// afterCreateCallback will invoke `AfterCreate`, `AfterSave` method after creating
func afterCreateCallback(scope *Scope) {
	if !scope.HasError() {
		scope.CallMethod("AfterCreate")
	}
	if !scope.HasError() {
		scope.CallMethod("AfterSave")
	}
}

func SetIdCallback(scope *Scope) {
	value := scope.Value
	reflectValue := reflect.ValueOf(value)

	for reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}

	pf := scope.PrimaryField()

	if pf != nil && pf.TagSettings.Flag("SERIAL") {
		i := pf.Field.Addr().Interface()
		if gen, ok := i.(Generator); ok {
			gen.Generate()
			pf.Set(reflect.ValueOf(gen).Elem())
		}
	}
}
