package aorm

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
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
	DefaultCallback.Create().Register("aorm:create_children", createChildrenCallback)
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
		if user, ok := scope.db.GetCurrentUser(); ok {
			scope.SetColumn("CreatedByID", RawOfId(user))
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
				} else if field.IsPrimaryKey {
					columns = append(columns, scope.Quote(field.DBName))
					placeholders = append(placeholders, scope.AddToVars(field.Field.Interface()))
				} else if !field.IsBlank || field.Flag.Has(FieldCreationStoreEmpty) {
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

	var lastInsertIDReturningSuffix string
	if returningColumn != "" {
		lastInsertIDReturningSuffix = scope.Dialect().LastInsertIDReturningSuffix(quotedTableName, returningColumn)
	}

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
			row := scope.SQLDB().QueryRow(scope.Query.Query, scope.Query.Args...)
			if err := row.Scan(primaryField.Field.Addr().Interface()); scope.Err(err) == nil {
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
		db := scope.db.New().Table(scope.TableName()).ModelStruct(scope.modelStruct, scope.db.Val)
		columns := blankColumnsWithDefaultValue.([]string)
		if id := scope.modelStruct.GetID(scope.Value); id != nil && !id.IsZero() {
			// skip id columns
			quote := func(c string) string { return Quote(scope.Dialect(), c) }
			for _, f := range id.Fields() {
				for i, c := range columns {
					if quote(f.DBName) == c {
						columns = append(columns[0:i], columns[i:len(columns)-1]...)
						break
					}
				}
			}
		}
		if len(columns) > 0 {
			db.Select(columns).Scan(scope.Value)
		}
	}
}

// createChildrenCallback will creates children
func createChildrenCallback(scope *Scope) {
	if scope.HasError() {
		return
	}

	if _, ok := scope.InstanceGet("aorm:skip_children_create"); ok {
		return
	}

	var id = scope.instance.ID()

	for _, child := range scope.modelStruct.Children {
		v := scope.instance.FieldsMap[child.ParentField.Name].Field
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				continue
			}
		} else {
			v2 := reflect.New(v.Type())
			v2.Elem().Set(v)
			v = v2
		}
		childScope := scope.db.NewScope(v.Interface())
		_, err := CopyIdTo(id, childScope.ID())
		if err != nil {
			scope.Err(errors.Wrap(err, "copy ID from parent to child"))
			return
		}

		childScope.modelStruct = child
		if scope.Err(childScope.callCallbacks(childScope.db.parent.callbacks.creates).db.Error) != nil {
			return
		}
	}
}

// afterCreateCallback will invoke `AfterCreate`, `AfterSave` method after creating
func afterCreateCallback(scope *Scope) {
	if scope.HasError() {
		return
	}

	scope.CallMethod("AfterCreate")

	if scope.HasError() {
		return
	}

	scope.CallMethod("AfterSave")
}

func SetIdCallback(scope *Scope) {
	if !scope.ID().IsZero() {
		return
	}

	value := scope.Value
	reflectValue := reflect.ValueOf(value)

	for reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}

	pf := scope.PrimaryField()

	if gen, ok := value.(IDGenerator); ok {
		val := gen.AormGenerateID().Values()[0].Raw()
		pf.Set(reflect.ValueOf(val).Elem())
	} else {
		if pf != nil && pf.TagSettings.Flag("SERIAL") {
			i := pf.Field.Addr().Interface()
			if gen, ok := i.(Generator); ok && gen.IsZero() {
				gen.Generate()
				pf.Set(reflect.ValueOf(gen).Elem())
			}
		}
	}
}
