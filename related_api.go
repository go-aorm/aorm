package aorm

type (
	OnRelated interface {
		AormOnRelated(fromScope, scope *Scope, db *DB) *DB
	}

	OnRelatedField interface {
		AormOnRelatedField(fromScope, scope *Scope, db *DB, field *Field) *DB
	}
)
