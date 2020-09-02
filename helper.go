package aorm

import "reflect"

const (
	SqlZeroString    = 'ø'
	SqlZeroByteArray = 'Ø'

	SqlStringOpen  = '«'
	SqlStringClose = '»'
)

func SqlZeroOf(typ reflect.Type) string {
	switch typ.Kind() {
	case reflect.String:
		return string(SqlZeroString)
	case reflect.Bool:
		return "FALSE"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return "0"
	case reflect.Slice, reflect.Array:
		// []byte
		if typ.Elem().Kind() == reflect.Uint8 {
			return string(SqlZeroByteArray)
		}
	}
	return ""
}
