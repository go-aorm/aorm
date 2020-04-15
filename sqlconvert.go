//go:generate sh -c "echo 'package sql\n\nvar (\n\tGithubComMoisespsenaGoAormConvertAssign = convertAssign\n\tGithubComMoisespsenaGoAormConvertAssignRows = convertAssignRows\n)' > \"$GOROOT/src/database/sql/GithubComMoisespsenaGoAorm.go\""

package aorm

import (
	"database/sql"
)

var (
	SqlConvertAssign     = sql.GithubComMoisespsenaGoAormConvertAssign
	SqlConvertAssignRows = sql.GithubComMoisespsenaGoAormConvertAssignRows
)
