package aorm

import (
	"database/sql"
)

type ColumnDismissibleTypeScanerHandler interface {
	Scaner(scope *Scope, record interface{}, column string) sql.Scanner
}
