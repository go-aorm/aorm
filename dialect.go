package aorm

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

var dialectsMap = map[string]Dialector{}

func newDialect(name string, db SQLCommon) (dialect Dialector) {
	defer func() {
		dialect.Init()
	}()
	if value, ok := dialectsMap[name]; ok {
		dialect = reflect.New(reflect.TypeOf(value).Elem()).Interface().(Dialector)
		dialect.SetDB(db)
		return dialect
	}

	fmt.Printf("`%v` is not officially supported, running under compatibility mode.\n", name)
	commontDialect := &commonDialect{}
	commontDialect.SetDB(db)
	return commontDialect
}

// RegisterDialect register new dialect
func RegisterDialect(name string, dialect Dialector) {
	dialectsMap[name] = dialect
}

// GetDialect gets the dialect for the specified dialect name
func GetDialect(name string) (dialect Dialector, ok bool) {
	dialect, ok = dialectsMap[name]
	return
}

// MustGetDialect gets the dialect for the specified dialect name
func MustGetDialect(name string) (dialect Dialector) {
	var ok bool
	if dialect, ok = dialectsMap[name]; !ok {
		panic(errors.New(fmt.Sprintf("dialect %q is not registered", name)))
	}
	return
}

type FieldStructure struct {
	Type         reflect.Type
	TagSettings  map[string]string
	Assigner     Assigner
	IsPrimaryKey bool
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
		size = 255
	}

	// Default type from tag setting
	additionalType = field.TagSettings["NOT NULL"] + " " + field.TagSettings["UNIQUE"]
	if value, ok := field.TagSettings["DEFAULT"]; ok {
		// not default expression
		if value != "DEFAULT" {
			additionalType = additionalType + " DEFAULT " + value
		}
	}

	return fieldValue, dataType, size, strings.TrimSpace(additionalType)
}

func currentDatabaseAndTable(dialect Dialector, tableName string) (string, string) {
	if strings.Contains(tableName, ".") {
		splitStrings := strings.SplitN(tableName, ".", 2)
		return splitStrings[0], splitStrings[1]
	}
	return dialect.CurrentDatabase(), tableName
}
