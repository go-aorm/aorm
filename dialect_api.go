package aorm

import "reflect"

// Dialector interface contains behaviors that differ across SQL database
type Dialector interface {
	Quoter
	KeyNamer

	// Init the dialect
	Init()

	// Cast sql cast's from to
	Cast(from, to string) string

	// GetName get dialect's name
	GetName() string

	// SetDB set db for dialect
	SetDB(db SQLCommon)

	// BindVar return the placeholder for actual values in SQL statements, in many dbs it is "?", Postgres using $1
	BindVar(i int) string

	// DataTypeOf return data's sql type
	DataTypeOf(field *FieldStructure) string

	// HasIndex check has index or not
	HasIndex(tableName string, indexName string) bool
	// HasForeignKey check has foreign key or not
	HasForeignKey(tableName string, foreignKeyName string) bool
	// RemoveIndex remove index
	RemoveIndex(tableName string, indexName string) error
	// HasTable check has table or not
	HasTable(tableName string) bool
	// HasColumn check has column or not
	HasColumn(tableName string, columnName string) bool
	// ModifyColumn modify column's type
	ModifyColumn(tableName string, columnName string, typ string) error

	// LimitAndOffsetSQL return generated SQL with Limit and Offset, as mssql has special case
	LimitAndOffsetSQL(limit, offset interface{}) string
	// SelectFromDummyTable return select values, for most dbs, `SELECT values` just works, mysql needs `SELECT value FROM DUAL`
	SelectFromDummyTable() string
	// LastInsertIdReturningSuffix most dbs support LastInsertId, but postgres needs to use `RETURNING`
	LastInsertIDReturningSuffix(tableName, columnName string) string
	// DefaultValueStr
	DefaultValueStr() string

	// CurrentDatabase return current database name
	CurrentDatabase() string

	Assigners() map[reflect.Type]Assigner
	RegisterAssigner(assigner ...Assigner)
	GetAssigner(typ reflect.Type) (assigner Assigner)
	DuplicateUniqueIndexError(indexes IndexMap, tableName string, sqlErr error) (err error)
	ZeroValueOf(typ reflect.Type) string
	BytesToSql(b []byte) string

	PrepareSQL(sql string) string
}
