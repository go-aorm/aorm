package aorm

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Define callbacks for updating
func init() {
	DefaultCallback.Update().Register("aorm:start", startUpdateCallback)
	DefaultCallback.Update().Register("aorm:assign_updating_attributes", assignUpdatingAttributesCallback)
	DefaultCallback.Update().Register("aorm:begin_transaction", beginTransactionCallback)
	DefaultCallback.Update().Register("aorm:before_update", beforeUpdateCallback)
	DefaultCallback.Update().Register("aorm:save_before_associations", saveBeforeAssociationsCallback)
	DefaultCallback.Update().Register("aorm:update_time_stamp", updateTimeStampForUpdateCallback)
	DefaultCallback.Update().Register("aorm:audited", auditedForUpdateCallback)
	DefaultCallback.Update().Register("aorm:update", updateCallback)
	DefaultCallback.Update().Register("aorm:save_after_associations", saveAfterAssociationsCallback)
	DefaultCallback.Update().Register("aorm:after_update", afterUpdateCallback)
	DefaultCallback.Update().Register("aorm:commit_or_rollback_transaction", commitOrRollbackTransactionCallback)
}

// startUpdateCallback starts the updating callbacks
func startUpdateCallback(scope *Scope) {
	scope.Operation = OpUpdate
}

// assignUpdatingAttributesCallback assign updating attributes to model
func assignUpdatingAttributesCallback(scope *Scope) {
	if attrs, ok := scope.InstanceGet("aorm:update_interface"); ok {
		if updateMaps, hasUpdate := scope.updatedAttrsWithValues(attrs); hasUpdate {
			scope.InstanceSet("aorm:update_attrs", updateMaps)
		} else {
			scope.SkipLeft()
		}
	}
}

// beforeUpdateCallback will invoke `BeforeSave`, `BeforeUpdate` method before updating
func beforeUpdateCallback(scope *Scope) {
	if scope.DB().HasBlockGlobalUpdate() && !scope.hasConditions() {
		scope.Err(errors.New("Missing WHERE clause while updating"))
		return
	}
	if _, ok := scope.Get("aorm:update_column"); !ok {
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
		if user := scope.GetCurrentUserID(); user != nil {
			scope.SetColumn("UpdatedByID", user)
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
			_ = scope.Err(ErrSingleUpdateKey)
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
