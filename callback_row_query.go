package aorm

import "database/sql"

// Define callbacks for row query
func init() {
	DefaultCallback.RowQuery().Register("gorm:row_query", rowQueryCallback)
	DefaultCallback.RowQuery().Before("gorm:row_query").Register("gorm:inline_preload", inlinePreloadCallback)
}

type RowQueryResult struct {
	Row *sql.Row
}

type RowsQueryResult struct {
	Rows  *sql.Rows
	Error error
}

// queryCallback used to query data from database
func rowQueryCallback(scope *Scope) {
	if result, ok := scope.InstanceGet("row_query_result"); ok {
		scope.prepareQuerySQL()
		scope.ExecTime = NowFunc()

		if rowResult, ok := result.(*RowQueryResult); ok {
			rowResult.Row = scope.runQueryRow()
		} else if rowsResult, ok := result.(*RowsQueryResult); ok {
			rowsResult.Rows = scope.runQueryRows()
			rowsResult.Error = scope.Error()
		}
	}
}
