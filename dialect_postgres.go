package aorm

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type postgres struct {
	commonDialect
}

func init() {
	RegisterDialect("postgres", &postgres{})
	RegisterDialect("cloudsqlpostgres", &postgres{})
}

func (postgres) GetName() string {
	return "postgres"
}

func (postgres) BindVar(i int) string {
	return "$" + strconv.Itoa(i)
}

func (postgres) Cast(from, to string) string {
	return from + "::" + to
}

func (this *postgres) DataTypeOf(field *FieldStructure) string {
	var dataValue, sqlType, size, additionalType = ParseFieldStructForDialect(field, this)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Bool:
			sqlType = "boolean"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uintptr:
			if this.fieldCanAutoIncrement(field) {
				field.TagSettings["AUTO_INCREMENT"] = "AUTO_INCREMENT"
				sqlType = "serial"
			} else {
				sqlType = "integer"
			}
		case reflect.Int64, reflect.Uint32, reflect.Uint64:
			if this.fieldCanAutoIncrement(field) {
				field.TagSettings["AUTO_INCREMENT"] = "AUTO_INCREMENT"
				sqlType = "bigserial"
			} else {
				sqlType = "bigint"
			}
		case reflect.Float32:
			sqlType = "float4"
		case reflect.Float64:
			sqlType = "float8"
		case reflect.String:
			if _, ok := field.TagSettings["SIZE"]; !ok {
				size = 0 // if SIZE haven'T been set, use `text` as the default type, as there are no performance different
			}

			if size > 0 && size < 65532 {
				sqlType = fmt.Sprintf("varchar(%d)", size)
			} else {
				sqlType = "text"
			}
		case reflect.Struct:
			if _, ok := dataValue.Interface().(time.Time); ok {
				sqlType = "timestamp with time zone"
			}
		case reflect.Map:
			if dataValue.Type().Name() == "Hstore" {
				sqlType = "hstore"
			}
		default:
			if IsByteArrayOrSlice(dataValue) {
				sqlType = "bytea"

				if isUUID(dataValue) {
					sqlType = "uuid"
				}

				if isJSON(dataValue) {
					sqlType = "jsonb"
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
		panic(fmt.Sprintf("invalid sql type %s (%s) for postgres", dataValue.Type().Name(), dataValue.Kind().String()))
	}

	if strings.TrimSpace(additionalType) == "" {
		return sqlType
	}
	return fmt.Sprintf("%v %v", sqlType, additionalType)
}

func (postgres) BytesToSql(b []byte) string {
	return `'\x` + hex.EncodeToString(b) + `'`
}

func (this postgres) HasIndex(tableName string, indexName string) bool {
	var count int
	this.db.QueryRow("SELECT count(*) FROM pg_indexes WHERE tablename = $1 AND indexname = $2 AND schemaname = CURRENT_SCHEMA()", tableName, indexName).Scan(&count)
	return count > 0
}

func (this postgres) HasForeignKey(tableName string, foreignKeyName string) bool {
	var count int
	this.db.QueryRow("SELECT count(con.conname) FROM pg_constraint con WHERE $1::regclass::oid = con.conrelid AND con.conname = $2 AND con.contype='f'", tableName, foreignKeyName).Scan(&count)
	return count > 0
}

func (this postgres) HasTable(tableName string) bool {
	var count int
	this.db.QueryRow("SELECT count(*) FROM INFORMATION_SCHEMA.tables WHERE table_name = $1 AND table_type = 'BASE TABLE' AND table_schema = CURRENT_SCHEMA()", tableName).Scan(&count)
	return count > 0
}

func (this postgres) HasColumn(tableName string, columnName string) bool {
	var count int
	this.db.QueryRow("SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_name = $1 AND column_name = $2 AND table_schema = CURRENT_SCHEMA()", tableName, columnName).Scan(&count)
	return count > 0
}

func (this postgres) CurrentDatabase() (name string) {
	this.db.QueryRow("SELECT CURRENT_DATABASE()").Scan(&name)
	return
}

func (this postgres) LastInsertIDReturningSuffix(tableName, key string) string {
	return fmt.Sprintf("RETURNING %v.%v", tableName, key)
}

func (postgres) SupportLastInsertID() bool {
	return false
}

func (this postgres) DuplicateUniqueIndexError(indexes IndexMap, tableName string, sqlErr error) (err error) {
	msg := sqlErr.Error()
	if strings.Contains(msg, "duplicate") {
		if pos := strings.IndexRune(msg, '"'); pos > 0 {
			index_name := msg[pos+1 : pos+strings.IndexRune(msg[pos+1:], '"')+1]
			if index := indexes.FromDbName(this, tableName, index_name); index != nil {
				return &DuplicateUniqueIndexError{index, sqlErr}
			}
		}
	}
	return sqlErr
}

func (this postgres) Init() {
	if this.db == nil {
		return
	}
	if _, err := this.db.Exec(pgbidFuncs); err != nil {
		panic(err)
	}
}

func isUUID(value reflect.Value) bool {
	if value.Kind() != reflect.Array || value.Type().Len() != 16 {
		return false
	}
	typename := value.Type().Name()
	lower := strings.ToLower(typename)
	return "uuid" == lower || "guid" == lower
}

func isJSON(value reflect.Value) bool {
	_, ok := value.Interface().(json.RawMessage)
	return ok
}

const (
	pgbidFuncs = `
CREATE OR REPLACE FUNCTION pgbid_get_utc_time(bid BYTEA)
RETURNS TIMESTAMP AS $$
SELECT to_timestamp(
        (get_byte(bid,0)<<24) +
        (get_byte(bid,1)<<16) +
        (get_byte(bid,2)<<8) +
        (get_byte(bid,3))
    )::TIMESTAMPTZ AT TIME ZONE 'UTC'
$$ LANGUAGE SQL IMMUTABLE STRICT;

CREATE OR REPLACE FUNCTION pgbid_get_utc_date(bid BYTEA) RETURNS DATE AS $$
SELECT public.pgbid_get_utc_time(bid)::DATE
$$ LANGUAGE SQL IMMUTABLE STRICT;

CREATE OR REPLACE FUNCTION pgbid_get_time(bid BYTEA)
  RETURNS TIMESTAMPTZ
AS $$
SELECT (public.pgbid_get_utc_time(bid) AT TIME ZONE 'UTC')::TIMESTAMPTZ
$$ LANGUAGE SQL IMMUTABLE STRICT;

CREATE OR REPLACE FUNCTION pgbid_get_date(bid BYTEA) RETURNS DATE AS $$
SELECT public.pgbid_get_time(bid)::DATE
$$ LANGUAGE SQL IMMUTABLE STRICT;

CREATE OR REPLACE FUNCTION pgbid_to_text(b BYTEA) RETURNS TEXT AS
$$
    SELECT rtrim(replace(replace(encode(b, 'base64'), '/', '_'), '+', '-'), '=');
$$ LANGUAGE SQL IMMUTABLE STRICT;

CREATE OR REPLACE FUNCTION pgbid_to_bytea(t TEXT) RETURNS BYTEA AS
$$
	-- Base64 encode length: (BYTEA_LENGTH + 2) / 3 * 4 -> (12 + 2) / 3 * 4 = 16
    SELECT decode(rpad(replace(replace(t, '_', '/'), '-', '+'), 16, '='), 'base64');
$$ LANGUAGE SQL IMMUTABLE STRICT;
`
)
