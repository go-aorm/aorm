package aorm

import (
	"database/sql"
	"reflect"
)

type (
	// SQLCommon is the minimal database connection functionality gorm requires.  Implemented by *sql.DB.
	SQLCommon interface {
		Exec(query string, args ...interface{}) (sql.Result, error)
		Prepare(query string) (*sql.Stmt, error)
		Query(query string, args ...interface{}) (*sql.Rows, error)
		QueryRow(query string, args ...interface{}) *sql.Row
	}

	sqlDb interface {
		Begin() (*sql.Tx, error)
	}

	sqlTx interface {
		Commit() error
		Rollback() error
	}

	DbDataTyper interface {
		AormDataType(dialect Dialector) string
	}

	FieldAssigner interface {
		AormAssigner() Assigner
	}

	ArgBinder interface {
		DbBindVar(dialect Dialector, argVar string) string
	}

	FieldTypeTagSettinger interface {
		TagSetting(field reflect.StructField) TagSetting
	}

	Generator interface {
		Zeroer
		Generate()
	}

	Clauser interface {
		Clause(scope *Scope) (result Query)
	}

	WhereClauser interface {
		WhereClause(scope *Scope) (result Query)
	}

	ClauseScoper interface {
		TableNamer
		ToVarsAppender
		QuotedTableName() string
		Struct() *ModelStruct
		Instance() *Instance
		ScopeOfField(fieldName string) *Scope
	}

	BytesParser interface {
		ParseBytes(b []byte) error
	}

	StringParser interface {
		ParseString(s string) (err error)
	}

	FieldsForUpdateAcceptor interface {
		AormAcceptFieldsForUpdate(scope *Scope) (fields []string, apply bool)
	}

	FieldsForUpdateExcluder interface {
		AormExcludeFieldsForUpdate(scope *Scope) (fields []string, apply bool)
	}

	FieldsForCreateAcceptor interface {
		AormAcceptFieldsForCreate(scope *Scope) (fields []string, apply bool)
	}

	FieldsForCreateExcluder interface {
		AormExcludeFieldsForCreate(scope *Scope) (fields []string, apply bool)
	}

	ToVarsAppender interface {
		AddToVars(value interface{}) (replacement string)
	}

	Selector interface {
		Select(scope *Scope, tableName string) Query
	}
)
