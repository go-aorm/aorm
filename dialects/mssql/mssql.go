package mssql

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	// Importing mssql driver package only in dialect file, otherwide not needed
	_ "github.com/denisenkom/go-mssqldb"

	"github.com/moisespsena-go/aorm"
)

func setIdentityInsert(scope *aorm.Scope) {
	if scope.Dialect().GetName() == "mssql" {
		for _, field := range scope.PrimaryFields() {
			if _, ok := field.TagSettings["AUTO_INCREMENT"]; ok && !field.IsBlank {
				scope.NewDB().Exec(fmt.Sprintf("SET IDENTITY_INSERT %v ON", scope.TableName()))
				scope.InstanceSet("mssql:identity_insert_on", true)
			}
		}
	}
}

func turnOffIdentityInsert(scope *aorm.Scope) {
	if scope.Dialect().GetName() == "mssql" {
		if _, ok := scope.InstanceGet("mssql:identity_insert_on"); ok {
			scope.NewDB().Exec(fmt.Sprintf("SET IDENTITY_INSERT %v OFF", scope.TableName()))
		}
	}
}

func init() {
	aorm.DefaultCallback.Create().After("aorm:begin_transaction").Register("mssql:set_identity_insert", setIdentityInsert)
	aorm.DefaultCallback.Create().Before("aorm:commit_or_rollback_transaction").Register("mssql:turn_off_identity_insert", turnOffIdentityInsert)
	aorm.RegisterDialect("mssql", &mssql{})
}

type mssql struct {
	db aorm.SQLCommon
	aorm.DefaultKeyNamer
}

func (s mssql) QuoteChar() rune {
	panic("implement me")
}

func (s mssql) Init() {
	panic("implement me")
}

func (s mssql) Cast(from, to string) string {
	panic("implement me")
}

func (s mssql) Assigners() map[reflect.Type]aorm.Assigner {
	panic("implement me")
}

func (s mssql) RegisterAssigner(assigner ...aorm.Assigner) {
	panic("implement me")
}

func (s mssql) GetAssigner(typ reflect.Type) (assigner aorm.Assigner) {
	panic("implement me")
}

func (s mssql) DuplicateUniqueIndexError(indexes aorm.IndexMap, tableName string, sqlErr error) (err error) {
	panic("implement me")
}

func (mssql) GetName() string {
	return "mssql"
}

func (s *mssql) SetDB(db aorm.SQLCommon) {
	s.db = db
}

func (mssql) BindVar(i int) string {
	return "$$$" // ?
}

func (mssql) Quote(key string) string {
	return fmt.Sprintf(`[%s]`, key)
}

func (s *mssql) DataTypeOf(field *aorm.FieldStructure) string {
	var dataValue, sqlType, size, additionalType = aorm.ParseFieldStructForDialect(field, s)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Bool:
			sqlType = "bit"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
			if s.fieldCanAutoIncrement(field) {
				field.TagSettings["AUTO_INCREMENT"] = "AUTO_INCREMENT"
				sqlType = "int IDENTITY(1,1)"
			} else {
				sqlType = "int"
			}
		case reflect.Int64, reflect.Uint64:
			if s.fieldCanAutoIncrement(field) {
				field.TagSettings["AUTO_INCREMENT"] = "AUTO_INCREMENT"
				sqlType = "bigint IDENTITY(1,1)"
			} else {
				sqlType = "bigint"
			}
		case reflect.Float32, reflect.Float64:
			sqlType = "float"
		case reflect.String:
			if size > 0 && size < 8000 {
				sqlType = fmt.Sprintf("nvarchar(%d)", size)
			} else {
				sqlType = "nvarchar(max)"
			}
		case reflect.Struct:
			if _, ok := dataValue.Interface().(time.Time); ok {
				sqlType = "datetimeoffset"
			}
		default:
			if aorm.IsByteArrayOrSlice(dataValue) {
				if size > 0 && size < 8000 {
					sqlType = fmt.Sprintf("varbinary(%d)", size)
				} else {
					sqlType = "varbinary(max)"
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
		panic(fmt.Sprintf("invalid sql type %s (%s) for mssql", dataValue.Type().Name(), dataValue.Kind().String()))
	}

	if strings.TrimSpace(additionalType) == "" {
		return sqlType
	}
	return fmt.Sprintf("%v %v", sqlType, additionalType)
}

func (s mssql) fieldCanAutoIncrement(field *aorm.FieldStructure) bool {
	if value, ok := field.TagSettings["AUTO_INCREMENT"]; ok {
		return value != "FALSE"
	}
	return field.IsPrimaryKey
}

func (s mssql) HasIndex(tableName string, indexName string) bool {
	var count int
	s.db.QueryRow("SELECT count(*) FROM sys.indexes WHERE name=? AND object_id=OBJECT_ID(?)", indexName, tableName).Scan(&count)
	return count > 0
}

func (s mssql) RemoveIndex(tableName string, indexName string) error {
	_, err := s.db.Exec(fmt.Sprintf("DROP INDEX %v ON %v", indexName, s.Quote(tableName)))
	return err
}

func (s mssql) HasForeignKey(tableName string, foreignKeyName string) bool {
	var count int
	currentDatabase, tableName := currentDatabaseAndTable(&s, tableName)
	s.db.QueryRow(`SELECT count(*) 
	FROM sys.foreign_keys as F inner join sys.tables as T on F.parent_object_id=T.object_id 
		inner join information_schema.tables as I on I.TABLE_NAME = T.name 
	WHERE F.name = ? 
		AND T.Name = ? AND I.TABLE_CATALOG = ?;`, foreignKeyName, tableName, currentDatabase).Scan(&count)
	return count > 0
}

func (s mssql) HasTable(tableName string) bool {
	var count int
	currentDatabase, tableName := currentDatabaseAndTable(&s, tableName)
	s.db.QueryRow("SELECT count(*) FROM INFORMATION_SCHEMA.tables WHERE table_name = ? AND table_catalog = ?", tableName, currentDatabase).Scan(&count)
	return count > 0
}

func (s mssql) HasColumn(tableName string, columnName string) bool {
	var count int
	currentDatabase, tableName := currentDatabaseAndTable(&s, tableName)
	s.db.QueryRow("SELECT count(*) FROM information_schema.columns WHERE table_catalog = ? AND table_name = ? AND column_name = ?", currentDatabase, tableName, columnName).Scan(&count)
	return count > 0
}

func (s mssql) ModifyColumn(tableName string, columnName string, typ string) error {
	_, err := s.db.Exec(fmt.Sprintf("ALTER TABLE %v ALTER COLUMN %v %v", tableName, columnName, typ))
	return err
}

func (s mssql) CurrentDatabase() (name string) {
	s.db.QueryRow("SELECT DB_NAME() AS [Current Database]").Scan(&name)
	return
}

func (mssql) LimitAndOffsetSQL(limit, offset interface{}) (sql string) {
	if offset != nil {
		if parsedOffset, err := strconv.ParseInt(fmt.Sprint(offset), 0, 0); err == nil && parsedOffset >= 0 {
			sql += fmt.Sprintf(" OFFSET %d ROWS", parsedOffset)
		}
	}
	if limit != nil {
		if parsedLimit, err := strconv.ParseInt(fmt.Sprint(limit), 0, 0); err == nil && parsedLimit >= 0 {
			if sql == "" {
				// add default zero offset
				sql += " OFFSET 0 ROWS"
			}
			sql += fmt.Sprintf(" FETCH NEXT %d ROWS ONLY", parsedLimit)
		}
	}
	return
}

func (mssql) SelectFromDummyTable() string {
	return ""
}

func (mssql) LastInsertIDReturningSuffix(tableName, columnName string) string {
	return ""
}

func (mssql) DefaultValueStr() string {
	return "DEFAULT VALUES"
}

func currentDatabaseAndTable(dialect aorm.Dialector, tableName string) (string, string) {
	if strings.Contains(tableName, ".") {
		splitStrings := strings.SplitN(tableName, ".", 2)
		return splitStrings[0], splitStrings[1]
	}
	return dialect.CurrentDatabase(), tableName
}

// JSON type to support easy handling of JSON data in character table fields
// using golang json.RawMessage for deferred decoding/encoding
type JSON struct {
	json.RawMessage
}

// Value get value of JSON
func (j JSON) Value() (driver.Value, error) {
	if len(j.RawMessage) == 0 {
		return nil, nil
	}
	return j.MarshalJSON()
}

// Scan scan value into JSON
func (j *JSON) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value (strcast):", value))
	}
	bytes := []byte(str)
	return json.Unmarshal(bytes, j)
}
