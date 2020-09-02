package aorm

import (
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func AbbrToTextType(abbr string) string {
	size := TextSize(abbr)
	if size == 0 {
		return "TEXT"
	}
	return fmt.Sprintf("VARCHAR(%d)", size)
}

func TextSize(typ string) uint16 {
	switch typ {
	case "tiny", "small":
		return 127
	case "medium":
		return 512
	case "large":
		return 1024
	default:
		if m := regexp.MustCompile(`(?i)^VARCHAR\s*\((\d+)\)`).FindAllStringSubmatch(typ, 1); m != nil {
			ui64, _ := strconv.ParseUint(m[0][1], 10, 16)
			return uint16(ui64)
		}
		return 0
	}
}

// IsByteArrayOrSlice returns true of the reflected value is an array or slice
func IsByteArrayOrSlice(value reflect.Value) bool {
	return (value.Kind() == reflect.Array || value.Kind() == reflect.Slice) && value.Type().Elem() == reflect.TypeOf(uint8(0))
}

// ParseFieldStructForDialect get field's sql data type
var ParseFieldStructForDialect = func(field *FieldStructure, dialect Dialector) (fieldValue reflect.Value, sqlType string, size int, additionalType string) {
	// Get redirected field type
	var (
		reflectType = field.Type
		dataType    = field.TagSettings["TYPE"]
	)

	for reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}

	if dataType != "" {
		switch reflectType.Kind() {
		case reflect.String:
			dataType = AbbrToTextType(dataType)
		}
	}

	if dataType == "" && field.Assigner != nil {
		dataType = field.Assigner.SQLType(dialect)
	}

	// Get redirected field value
	fieldValue = reflect.Indirect(reflect.New(reflectType))

	if dataTyper, ok := fieldValue.Interface().(DbDataTyper); ok {
		dataType = dataTyper.AormDataType(dialect)
	}

	// Get scanner's real value
	if dataType == "" {
		var getScannerValue func(reflect.Value)
		getScannerValue = func(value reflect.Value) {
			fieldValue = value
			if _, isScanner := reflect.New(fieldValue.Type()).Interface().(sql.Scanner); isScanner && fieldValue.Kind() == reflect.Struct {
				getScannerValue(fieldValue.Field(0))
			}
		}
		getScannerValue(fieldValue)
	}

	// Default Size
	if num, ok := field.TagSettings["SIZE"]; ok {
		size, _ = strconv.Atoi(num)
	} else if sizer, ok := field.Assigner.(SQLSizer); ok {
		size = sizer.SQLSize(dialect)
	} else {
		if ui16 := TextSize(dataType); size == 0 {
			size = 255
		} else {
			size = int(ui16)
		}
	}

	// Default type from tag setting
	additionalType = field.TagSettings["NOT NULL"] + " " + field.TagSettings["UNIQUE"]
	if value, ok := field.TagSettings["DEFAULT"]; ok {
		// not default expression
		if value == "DEFAULT" {
			var defaul = ""
			if ti, ok := reflect.New(reflectType).Interface().(DefaultDbValuer); ok {
				defaul = ti.AormDefaultDbValue(dialect)
			}
			if defaul == "" {
				defaul = dialect.ZeroValueOf(reflectType)
			}
			additionalType = additionalType + " DEFAULT " + defaul
		} else {
			additionalType = additionalType + " DEFAULT " + value
		}
	}

	return fieldValue, dataType, size, strings.TrimSpace(additionalType)
}

func DialectVarsReplacer(dialect Dialector, dest *[]interface{}) func(arg interface{}) (replacement string) {
	return func(value interface{}) (replacement string) {
		replacement = dialect.BindVar(len(*dest) + 1)
		return replacement
	}
}
