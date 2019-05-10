//go:generate go run build-hooks/pre-build/export-sql-funcs/cli/main.go

package aorm

import (
	"database/sql"
)

var (
	SqlConvertAssign     = sql.GithubComMoisespsenaGoAormConvertAssign
	SqlConvertAssignRows = sql.GithubComMoisespsenaGoAormConvertAssignRows
)
