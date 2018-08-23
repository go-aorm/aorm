package aorm

import (
	"errors"
	"fmt"
)

// Define callbacks for deleting
func init() {
	DefaultCallback.Delete().Register("gorm:begin_transaction", beginTransactionCallback)
	DefaultCallback.Delete().Register("gorm:before_delete", beforeDeleteCallback)
	DefaultCallback.Delete().Register("gorm:audited", auditedForDeleteCallback)
	DefaultCallback.Delete().Register("gorm:delete", deleteCallback)
	DefaultCallback.Delete().Register("gorm:after_delete", afterDeleteCallback)
	DefaultCallback.Delete().Register("gorm:commit_or_rollback_transaction", commitOrRollbackTransactionCallback)
}

// beforeDeleteCallback will invoke `BeforeDelete` method before deleting
func beforeDeleteCallback(scope *Scope) {
	if scope.DB().HasBlockGlobalUpdate() && !scope.hasConditions() {
		scope.Err(errors.New("Missing WHERE clause while deleting"))
		return
	}
	if !scope.HasError() {
		scope.CallMethod("BeforeDelete")
	}
}

// auditedForDeleteCallback will set `DeletedBy` when soft delete
func auditedForDeleteCallback(scope *Scope) {
	if _, ok := scope.Get("gorm:deleted_by_column"); !ok {
		if _, hasDeletedAtField := scope.FieldByName("DeletedAt"); hasDeletedAtField {
			if user, ok := getCurrentUser(scope); ok {
				if attrs, ok := scope.InstanceGet("gorm:update_attrs"); ok {
					updateAttrs := attrs.(map[string]interface{})
					updateAttrs["deleted_by"] = user
					scope.InstanceSet("gorm:update_attrs", updateAttrs)
				} else {
					scope.SetColumn("DeletedBy", user)
				}
			}
		}
	}
}

// deleteCallback used to delete data from database or set deleted_at to current time (when using with soft delete)
func deleteCallback(scope *Scope) {
	if !scope.HasError() {
		var extraOption string
		if str, ok := scope.Get("gorm:delete_option"); ok {
			extraOption = fmt.Sprint(str)
		}

		deletedAtField, hasDeletedAtField := scope.FieldByName("DeletedAt")

		if !scope.Search.Unscoped && hasDeletedAtField {
			scope.Raw(fmt.Sprintf(
				"UPDATE %v SET %v=%v%v%v",
				scope.QuotedTableName(),
				scope.Quote(deletedAtField.DBName),
				scope.AddToVars(NowFunc()),
				addExtraSpaceIfExist(scope.CombinedConditionSql()),
				addExtraSpaceIfExist(extraOption),
			)).Exec()
		} else {
			scope.Raw(fmt.Sprintf(
				"DELETE FROM %v%v%v",
				scope.QuotedTableName(),
				addExtraSpaceIfExist(scope.CombinedConditionSql()),
				addExtraSpaceIfExist(extraOption),
			)).Exec()
		}
	}
}

// afterDeleteCallback will invoke `AfterDelete` method after deleting
func afterDeleteCallback(scope *Scope) {
	if !scope.HasError() {
		scope.CallMethod("AfterDelete")
	}
}
