package aorm

import "sync"

var fakeDBMap sync.Map

// CreateFakeDB create a new fake db with dialect
func CreateFakeDB(dialect string) (db *DB) {
	db = &DB{
		values:  map[interface{}]interface{}{},
		dialect: newDialect(dialect, nil),
	}
	db.parent = db
	fakeDBMap.Store(dialect, db)
	return
}

// FakeDB get or create a new fake db with dialect
func FakeDB(dialect string) (db *DB) {
	dbi, ok := fakeDBMap.Load(dialect)
	if ok {
		db = dbi.(*DB)
	} else {
		db = CreateFakeDB(dialect)
		fakeDBMap.Store(dialect, db)
	}
	return
}
