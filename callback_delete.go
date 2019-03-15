package aorm

import (
	"errors"
	"fmt"
	"strings"
)

// Define callbacks for deleting
func init() {
	DefaultCallback.Delete().Register("gorm:begin_transaction", beginTransactionCallback)
	DefaultCallback.Delete().Register("gorm:before_delete", beforeDeleteCallback)
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

// deleteCallback used to delete data from database or set deleted_at to current time (when using with soft delete)
func deleteCallback(scope *Scope) {
	if !scope.HasError() {
		var extraOption string
		if str, ok := scope.Get("gorm:delete_option"); ok {
			extraOption = fmt.Sprint(str)
		}

		deletedAtField, hasDeletedAtField := scope.FieldByName(SoftDeleteFieldDeletedAt)

		if !scope.Search.Unscoped && hasDeletedAtField {
			var (
				pairs   []string
				columns = []string{deletedAtField.DBName}
				values  = []string{scope.AddToVars(NowFunc())}
			)

			if _, ok := scope.FieldByName(SoftDeleteFieldDeletedByID); ok {
				if user, ok := scope.GetCurrentUserID(); ok {
					columns = append(columns, SoftDeletedColumnDeletedByID)
					values = append(values, scope.AddToVars(user))
					scope.SetColumn(SoftDeleteFieldDeletedByID, user)
				}
			}

			for i, column := range columns {
				pairs = append(pairs, fmt.Sprintf("%v=%v", scope.Quote(column), values[i]))
			}

			scope.Raw(fmt.Sprintf(
				"UPDATE %v SET %v%v%v",
				scope.QuotedTableName(),
				strings.Join(pairs, ", "),
				addExtraSpaceIfExist(scope.CombinedConditionSql()),
				addExtraSpaceIfExist(extraOption),
			)).log(LOG_DELETE).Exec()
		} else {
			scope.Raw(fmt.Sprintf(
				"DELETE FROM %v%v%v",
				scope.QuotedTableName(),
				addExtraSpaceIfExist(scope.CombinedConditionSql()),
				addExtraSpaceIfExist(extraOption),
			)).log(LOG_DELETE).Exec()
		}
	}
}

// afterDeleteCallback will invoke `AfterDelete` method after deleting
func afterDeleteCallback(scope *Scope) {
	if !scope.HasError() {
		scope.CallMethod("AfterDelete")
	}
}
