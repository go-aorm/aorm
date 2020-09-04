package aorm

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Define callbacks for updating
func init() {
	DefaultCallback.Update().Register("aorm:start", startUpdateCallback)
	DefaultCallback.Update().Register("aorm:begin_transaction", beginTransactionCallback)
	DefaultCallback.Update().Register("aorm:before_update", beforeUpdateCallback)
	DefaultCallback.Update().Register("aorm:save_before_associations", saveBeforeAssociationsCallback)
	DefaultCallback.Update().Register("aorm:update_time_stamp", updateTimeStampForUpdateCallback)
	DefaultCallback.Update().Register("aorm:audited", auditedForUpdateCallback)
	DefaultCallback.Update().Register("aorm:update", updateCallback)
	DefaultCallback.Update().Register("aorm:update_children", updateChildrenCallback)
	DefaultCallback.Update().Register("aorm:save_after_associations", saveAfterAssociationsCallback)
	DefaultCallback.Update().Register("aorm:after_update", afterUpdateCallback)
	DefaultCallback.Update().Register("aorm:commit_or_rollback_transaction", commitOrRollbackTransactionCallback)
}

// startUpdateCallback starts the updating callbacks
func startUpdateCallback(scope *Scope) {
	scope.Operation = OpUpdate
}

// beforeUpdateCallback will invoke `BeforeSave`, `BeforeUpdate` method before updating
func beforeUpdateCallback(scope *Scope) {
	if scope.DB().HasBlockGlobalUpdate() && !scope.hasConditions() {
		scope.Err(errors.New("Missing WHERE clause while updating"))
		return
	}
	if attrs, ok := scope.InstanceGet("aorm:update_interface"); ok {
		if !scope.HasError() {
			scope.CallMethod("BeforeSave")
		}
		if !scope.HasError() {
			scope.CallMethod("BeforeUpdate")
		}
		if updateMaps, hasUpdate := scope.updatedAttrsWithValues(attrs); hasUpdate {
			scope.InstanceSet("aorm:update_attrs", updateMaps)
		} else {
			scope.SkipLeft()
		}
	} else if _, ok := scope.Get("aorm:update_column"); !ok {
		if !scope.HasError() {
			scope.CallMethod("BeforeSave")
		}
		if !scope.HasError() {
			scope.CallMethod("BeforeUpdate")
		}
	}
}

// updateTimeStampForUpdateCallback will set `UpdatedAt` when updating
func updateTimeStampForUpdateCallback(scope *Scope) {
	if _, ok := scope.Get("aorm:updated_at_column"); !ok {
		scope.SetColumn("UpdatedAt", NowFunc())
	}
}

// auditedForUpdateCallback will set `UpdatedByID` when updating
func auditedForUpdateCallback(scope *Scope) {
	if _, ok := scope.Get("aorm:updated_by_column"); !ok {
		if user, ok := scope.db.GetCurrentUser(); ok {
			scope.SetColumn("UpdatedByID", RawOfId(user))
		}
	}
}

// updateCallback the callback used to update data to database
func updateCallback(scope *Scope) {
	var sqls []string

	if updateAttrs, ok := scope.InstanceGet("aorm:update_attrs"); ok {
		// Sort the column names so that the generated SQL is the same every time.
		updateMap := updateAttrs.(map[string]interface{})
		var columns []string
		for c := range updateMap {
			columns = append(columns, c)
		}
		sort.Strings(columns)

		for _, column := range columns {
			value := updateMap[column]
			sqls = append(sqls, fmt.Sprintf("%v = %v", scope.Quote(column), scope.AddToVars(value)))
		}
	} else {
		for _, field := range scope.Instance().Fields {
			if scope.changeableField(field) {
				if !field.IsPrimaryKey && field.IsNormal {
					sqls = append(sqls, fmt.Sprintf("%v = %v", scope.Quote(field.DBName), scope.AddToVars(field.Field.Interface())))
				} else if relationship := field.Relationship; relationship != nil && relationship.Kind == "belongs_to" {
					for _, foreignKey := range relationship.ForeignDBNames {
						if foreignField, ok := scope.FieldByName(foreignKey); ok && !scope.changeableField(foreignField) {
							sqls = append(sqls,
								fmt.Sprintf("%v = %v", scope.Quote(foreignField.DBName), scope.AddToVars(foreignField.Field.Interface())))
						}
					}
				}
			}
		}
	}

	var extraOption string
	if str, ok := scope.Get("aorm:update_option"); ok {
		extraOption = fmt.Sprint(str)
	}

	if len(sqls) == 0 {
		return
	}

	if len(scope.PrimaryFields()) > 0 && scope.PrimaryKeyZero() {
		if scope.db.SingleUpdate() {
			scope.Err(ErrSingleUpdateKey)
			return
		}
	}

	query := fmt.Sprintf(
		"UPDATE %v SET %v%v%v",
		scope.QuotedTableName(),
		strings.Join(sqls, ", "),
		addExtraSpaceIfExist(scope.CombinedConditionSql()),
		addExtraSpaceIfExist(extraOption),
	)
	scope.Raw(query)

	if scope.checkDryRun() {
		return
	}
	scope.log(LOG_UPDATE).Exec()
}

// updateChildrenCallback
func updateChildrenCallback(scope *Scope) {
	if _, ok := scope.Get("aorm:update_column"); !ok && scope.db.RowsAffected == 1 {
		var id = scope.instance.ID()
		for _, child := range scope.modelStruct.Children {
			v := scope.instance.FieldsMap[child.ParentField.Name].Field
			if v.Kind() != reflect.Ptr {
				v = v.Addr()
			} else if v.IsNil() {
				childScope := scope.db.NewModelScope(child, child.Value)
				if _, err := CopyIdTo(id, childScope.ID()); err != nil {
					scope.Err(err)
					return
				}
				if err := scope.Err(childScope.callCallbacks(scope.db.callbacks.deletes).Error()); err != nil {
					return
				}
				continue
			}
			var newScope = func() *Scope {
				childScope := scope.db.NewModelScope(child, v.Interface())
				if _, err := CopyIdTo(id, childScope.ID()); err != nil {
					scope.Err(err)
				}
				return childScope.Set("aorm:disable_scope_transaction", true)
			}
			childScope := newScope()
			if scope.HasError() {
				return
			}

			newDB := childScope.callCallbacks(childScope.db.parent.callbacks.updates).db
			if newDB.Error == nil && newDB.RowsAffected == 0 {
				childScope := newScope()
				childScope.Search.Limit(1)
				if result := childScope.inlineCondition(id).callCallbacks(childScope.db.parent.callbacks.queries).db; result.Error != nil {
					if result.RecordNotFound() {
						newDB = newScope().callCallbacks(childScope.db.parent.callbacks.creates).db
					} else {
						scope.Err(result.Error)
					}
				}
			}
			if scope.Err(newDB.Error) != nil {
				return
			}
		}
	}
}

// afterUpdateCallback will invoke `AfterUpdate`, `AfterSave` method after updating
func afterUpdateCallback(scope *Scope) {
	if _, ok := scope.Get("aorm:update_column"); !ok {
		if !scope.HasError() {
			scope.CallMethod("AfterUpdate")
		}
		if !scope.HasError() {
			scope.CallMethod("AfterSave")
		}
	}
}
