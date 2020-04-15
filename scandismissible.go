package aorm

import "reflect"

type ColumnDismissibleTypeScaner struct {
	handlers map[reflect.Type]ColumnDismissibleTypeScanerHandler
}
