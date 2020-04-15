package aorm

import (
	"database/sql"
	"reflect"
)

func (scope *Scope) scan(rows *sql.Rows, columns []string, fields []*Field, result interface{}) {
	var (
		ignored            interface{}
		values             []interface{}
		selectFields       []*Field
		selectedColumnsMap = map[string]int{}
		scannerFields      = make(map[int]*Field)
		extraValues        []interface{}
		extraFieldsValues  []interface{}
		extraValuesSize    int
		extraFieldsSize    int
	)

	if scope.Search.extraSelects != nil {
		extraValuesSize = scope.Search.extraSelects.Size
		extraValues = scope.Search.extraSelects.NewValues()
	}

	if scope.Search.extraSelectsFields != nil {
		extraFieldsSize = scope.Search.extraSelectsFields.Size
		extraFieldsValues = scope.Search.extraSelectsFields.NewValues()
	}

	values = make([]interface{}, len(columns)-extraValuesSize-extraFieldsSize)

	for index, column := range columns[0:len(values)] {
		if scope.Search.defaultColumnValue != nil {
			if value := scope.Search.defaultColumnValue(scope, result, column); value != nil {
				values[index] = value
			} else {
				values[index] = &ignored
			}
		} else {
			values[index] = &ignored
		}

		selectFields = fields
		if idx, ok := selectedColumnsMap[column]; ok {
			selectFields = selectFields[idx+1:]
		}

		for fieldIndex, field := range selectFields {
			if field.DBName == column {
				scannerFields[index] = field
				fs := field.Scaner(scope.db.dialect)
				values[index] = fs
				selectedColumnsMap[column] = fieldIndex

				if field.IsNormal || field.IsReadOnly {
					break
				}
			}
		}
	}

	if err := scope.Err(rows.Scan(append(append(values, extraValues...), extraFieldsValues...)...)); err != nil {
		return
	}

	if scope.Search.columnsScannerCallback != nil {
		scope.Search.columnsScannerCallback(scope, result, columns[0:len(values)], values)
	}

	disableScanField := make(map[int]bool)

	if result != nil {
		if extraValuesSize > 0 {
			extraColumns := columns[len(values) : len(values)+extraFieldsSize]
			extraResult := make(map[string]*ExtraResult)
			for _, es := range scope.Search.extraSelects.Items {
				r := &ExtraResult{es, extraValues[:len(es.Values)], extraColumns[:len(es.Values)], make(map[string]int)}
				for i, name := range r.Names {
					r.Map[name] = i
					if !es.Ptrs[i] {
						r.Values[i] = reflect.Indirect(reflect.ValueOf(r.Values[i])).Interface()
					}
				}
				extraResult[es.key] = r
				extraValues = extraValues[len(es.Values)-1:]
			}
			if sev, ok := result.(ExtraSelectInterface); ok {
				sev.SetAormExtraScannedValues(extraResult)
			}

			for _, cb := range scope.Search.extraSelects.Callbacks {
				cb(result, extraResult)
			}
		}
		if extraFieldsSize > 0 {
			scope.Search.extraSelectsFields.SetValues(scope, result, extraFieldsValues)
		}

		scope.afterScanCallback(scannerFields, disableScanField)
		scope.callMethod("AormAfterScan", reflect.ValueOf(result))
	}
}
