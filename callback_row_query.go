package aorm

import "database/sql"

// Define callbacks for row query
func init() {
	DefaultCallback.RowQuery().Register("aorm:row_query", rowQueryCallback)
	DefaultCallback.RowQuery().Before("aorm:row_query").Register("aorm:inline_preload", inlinePreloadCallback)
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
	scope.prepareQuerySQL()
	if scope.HasError() {
		return
	}
	scope.ExecTime = NowFunc()

	if scope.checkDryRun() {
		return
	}
	if result, ok := scope.InstanceGet("row_query_result"); ok {
		switch t := result.(type) {
		case *RowQueryResult:
			t.Row = scope.runQueryRow()
		case *RowsQueryResult:
			t.Rows = scope.runQueryRows()
			t.Error = scope.Error()
		}
	}
}
