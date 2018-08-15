package gorm

type OnRelated interface {
	GormOnRelated(fromScope, scope *Scope, db *DB) *DB
}

type OnRelatedField interface {
	GormOnRelated(fromScope, scope *Scope, db *DB, field *Field) *DB
}
