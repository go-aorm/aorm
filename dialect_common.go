package aorm

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type commonDialect struct {
	db SQLCommon
	DefaultKeyNamer
	assigners map[reflect.Type]Assigner
}

func init() {
	RegisterDialect("common", &commonDialect{})
}

func (c *commonDialect) Assigners() map[reflect.Type]Assigner {
	return c.assigners
}

func (c *commonDialect) RegisterAssigner(assigners ...Assigner) {
	if c.assigners == nil {
		c.assigners = map[reflect.Type]Assigner{}
	}
	for _, assigner := range assigners {
		c.assigners[assigner.Type()] = assigner
	}
}

func (c *commonDialect) GetAssigner(typ reflect.Type) (assigner Assigner) {
	if c.assigners != nil {
		assigner = c.assigners[indirectType(typ)]
	}
	return
}

func (commonDialect) Init() {}

func (commonDialect) Cast(from, to string) string {
	return "CAST(" + from + "," + to + ")"
}

func (commonDialect) GetName() string {
	return "common"
}

func (s *commonDialect) SetDB(db SQLCommon) {
	s.db = db
}

func (commonDialect) BindVar(i int) string {
	return "$$$" // ?
}

func (commonDialect) QuoteChar() rune {
	return '"'
}

func (s *commonDialect) fieldCanAutoIncrement(field *FieldStructure) bool {
	if value, ok := field.TagSettings["AUTO_INCREMENT"]; ok {
		return strings.ToLower(value) != "false"
	}
	return field.IsPrimaryKey
}

func (s *commonDialect) DataTypeOf(field *FieldStructure) string {
	var dataValue, sqlType, size, additionalType = ParseFieldStructForDialect(field, s)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Bool:
			sqlType = "BOOLEAN"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
			if s.fieldCanAutoIncrement(field) {
				sqlType = "INTEGER AUTO_INCREMENT"
			} else {
				sqlType = "INTEGER"
			}
		case reflect.Int64, reflect.Uint64:
			if s.fieldCanAutoIncrement(field) {
				sqlType = "BIGINT AUTO_INCREMENT"
			} else {
				sqlType = "BIGINT"
			}
		case reflect.Float32, reflect.Float64:
			sqlType = "FLOAT"
		case reflect.String:
			if size > 0 && size < 65532 {
				sqlType = fmt.Sprintf("VARCHAR(%d)", size)
			} else {
				sqlType = "VARCHAR(65532)"
			}
		case reflect.Struct:
			if _, ok := dataValue.Interface().(time.Time); ok {
				sqlType = "TIMESTAMP"
			}
		default:
			if _, ok := dataValue.Interface().([]byte); ok {
				if size > 0 && size < 65532 {
					sqlType = fmt.Sprintf("BINARY(%d)", size)
				} else {
					sqlType = "BINARY(65532)"
				}
			}
		}
	} else if size > 0 {
		switch sqlType {
		case "CHAR", "VARCHAR":
			sqlType += fmt.Sprintf("(%d)", size)
		}
	}

	if sqlType == "" {
		panic(fmt.Sprintf("invalid sql type %s (%s) for commonDialect", dataValue.Type().Name(), dataValue.Kind().String()))
	}

	if strings.TrimSpace(additionalType) == "" {
		return sqlType
	}
	return fmt.Sprintf("%v %v", sqlType, additionalType)
}

func (s commonDialect) HasIndex(tableName string, indexName string) bool {
	var count int
	currentDatabase, tableName := currentDatabaseAndTable(&s, tableName)
	s.db.QueryRow("SELECT count(*) FROM INFORMATION_SCHEMA.STATISTICS WHERE table_schema = ? AND table_name = ? AND index_name = ?", currentDatabase, tableName, indexName).Scan(&count)
	return count > 0
}

func (s commonDialect) RemoveIndex(tableName string, indexName string) error {
	_, err := s.db.Exec(fmt.Sprintf("DROP INDEX %v", indexName))
	return err
}

func (s commonDialect) HasForeignKey(tableName string, foreignKeyName string) bool {
	return false
}

func (s commonDialect) HasTable(tableName string) bool {
	var count int
	currentDatabase, tableName := currentDatabaseAndTable(&s, tableName)
	s.db.QueryRow("SELECT count(*) FROM INFORMATION_SCHEMA.TABLES WHERE table_schema = ? AND table_name = ?", currentDatabase, tableName).Scan(&count)
	return count > 0
}

func (s commonDialect) HasColumn(tableName string, columnName string) bool {
	var count int
	currentDatabase, tableName := currentDatabaseAndTable(&s, tableName)
	s.db.QueryRow("SELECT count(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE table_schema = ? AND table_name = ? AND column_name = ?", currentDatabase, tableName, columnName).Scan(&count)
	return count > 0
}

func (s commonDialect) ModifyColumn(tableName string, columnName string, typ string) error {
	_, err := s.db.Exec(fmt.Sprintf("ALTER TABLE %v ALTER COLUMN %v TYPE %v", tableName, columnName, typ))
	return err
}

func (s commonDialect) CurrentDatabase() (name string) {
	s.db.QueryRow("SELECT DATABASE()").Scan(&name)
	return
}

func (commonDialect) LimitAndOffsetSQL(limit, offset interface{}) (sql string) {
	if limit != nil {
		if parsedLimit, err := strconv.ParseInt(fmt.Sprint(limit), 0, 0); err == nil && parsedLimit >= 0 {
			sql += fmt.Sprintf(" LIMIT %d", parsedLimit)
		}
	}
	if offset != nil {
		if parsedOffset, err := strconv.ParseInt(fmt.Sprint(offset), 0, 0); err == nil && parsedOffset >= 0 {
			sql += fmt.Sprintf(" OFFSET %d", parsedOffset)
		}
	}
	return
}

func (commonDialect) SelectFromDummyTable() string {
	return ""
}

func (commonDialect) LastInsertIDReturningSuffix(tableName, columnName string) string {
	return ""
}

func (commonDialect) DefaultValueStr() string {
	return "DEFAULT VALUES"
}

func (d commonDialect) DuplicateUniqueIndexError(_ IndexMap, _ string, sqlErr error) error {
	return sqlErr
}

// IsByteArrayOrSlice returns true of the reflected value is an array or slice
func IsByteArrayOrSlice(value reflect.Value) bool {
	return (value.Kind() == reflect.Array || value.Kind() == reflect.Slice) && value.Type().Elem() == reflect.TypeOf(uint8(0))
}
