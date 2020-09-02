package aorm

import (
	"fmt"
	"reflect"
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

func currentDatabaseAndTable(dialect Dialector, tableName string) (string, string) {
	if strings.Contains(tableName, ".") {
		splitStrings := strings.SplitN(tableName, ".", 2)
		return splitStrings[0], splitStrings[1]
	}
	return dialect.CurrentDatabase(), tableName
}
