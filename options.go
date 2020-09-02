package aorm

const (
	OptKeyStoreBlankField = "aorm:store_blank_field"

	OptKeySkipPreload = "aorm:skip_preload"
	OptKeyAutoPreload = "aorm:auto_preload"

	OptKeySingleUpdate = "aorm:single_update"

	OptKeyCommitDisabled = "aorm:commit_disabled"
)

type Opt interface {
	Apply(db *DB) *DB
}

type OptFunc func(db *DB) *DB

func (o OptFunc) Apply(db *DB) *DB {
	return o(db)
}

func OptSkipPreload() Opt {
	return OptFunc(func(db *DB) *DB {
		return db.Set(OptKeySkipPreload, true)
	})
}

func OptStoreBlankField() Opt {
	return OptFunc(func(db *DB) *DB {
		return db.Set(OptKeyStoreBlankField, true)
	})
}

func OptAutoPreload() Opt {
	return OptFunc(func(db *DB) *DB {
		return db.Set(OptKeyAutoPreload, true)
	})
}

func OptForceSingleUpdate() Opt {
	return OptFunc(func(db *DB) *DB {
		return db.Set(OptKeySingleUpdate, false)
	})
}

func OptCommitDisabled() Opt {
	return OptFunc(func(db *DB) *DB {
		return db.Set(OptKeyCommitDisabled, true)
	})
}
