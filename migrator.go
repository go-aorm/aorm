package aorm

import "database/sql"

type Migrator struct {
	db           *DB
	failed       bool
	postHandlers []func(db *DB) error
	transaction  bool
}

func NewMigrator(db *DB) *Migrator {
	m := &Migrator{db: db}
	db.migrator = m
	return m
}

func (this *Migrator) Db() *DB {
	return this.db
}

func (this *Migrator) AutoMigrate(value ...interface{}) error {
	if err := this.db.autoMigrate(value...).Error; err != nil {
		this.failed = true
		return err
	}
	return nil
}

func (this *Migrator) PostHandler(f ...func(db *DB) error) {
	this.postHandlers = append(this.postHandlers, f...)
}

func (this *Migrator) Close() (err error) {
	if !this.failed {
		for _, h := range this.postHandlers {
			if err = h(this.db); err != nil {
				this.failed = true
				return
			}
		}
	}
	if this.transaction {
		if err = this.db.Commit().Error; err != nil {
			this.failed = true
			this.db.Rollback()
			return
		}
	}
	this.db.migrator = nil
	this.db = nil
	return
}

func (this *Migrator) Migrate(f ...func() error) (err error) {
	this.transaction = false
	if _, ok := this.db.CommonDB().(*sql.Tx); !ok {
		this.db = this.db.Begin()
		this.transaction = true
	}
	for _, f := range f {
		if err = f(); err != nil {
			return
		}
	}
	return this.Close()
}
