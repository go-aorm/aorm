package aorm

// CreateTable create table for models
func (s *DB) CreateTable(models ...interface{}) *DB {
	db := s.Unscoped()
	for _, model := range models {
		db = db.NewScope(model).createTable().db
	}
	return db
}

// DropTable drop table for models
func (s *DB) DropTable(values ...interface{}) *DB {
	db := s.clone()
	for _, value := range values {
		if tableName, ok := value.(string); ok {
			db = db.Table(tableName)
		}

		db = db.NewScope(value).dropTable().db
	}
	return db
}

// DropTableIfExists drop table if it is exist
func (s *DB) DropTableIfExists(values ...interface{}) *DB {
	db := s.clone()
	for _, value := range values {
		if s.HasTable(value) {
			db.AddError(s.DropTable(value).Error)
		}
	}
	return db
}

// HasTable check has table or not
func (s *DB) HasTable(value interface{}) bool {
	var (
		scope     = s.NewScope(value)
		tableName string
	)

	if name, ok := value.(string); ok {
		tableName = name
	} else {
		tableName = scope.TableName()
	}

	has := scope.Dialect().HasTable(tableName)
	s.AddError(scope.db.Error)
	return has
}

// AutoMigrate run auto migration for given models, will only add missing fields, won'T delete/change current data
func (s *DB) AutoMigrate(values ...interface{}) *DB {
	db := s.Unscoped()
	for _, value := range values {
		db = db.NewScope(value).autoMigrate().db
	}
	return db
}

// ModifyColumn modify column to type
func (s *DB) ModifyColumn(column string, typ string) *DB {
	scope := s.NewScope(s.Val)
	scope.modifyColumn(column, typ)
	return scope.db
}

// DropColumn drop a column
func (s *DB) DropColumn(column string) *DB {
	scope := s.NewScope(s.Val)
	scope.dropColumn(column)
	return scope.db
}

// AddIndex add index for columns with given name
func (s *DB) AddIndex(indexName string, columns ...string) *DB {
	scope := s.Unscoped().NewScope(s.Val)
	scope.addIndex(false, indexName, columns...)
	return scope.db
}

// AddUniqueIndex add unique index for columns with given name
func (s *DB) AddUniqueIndex(indexName string, columns ...string) *DB {
	scope := s.Unscoped().NewScope(s.Val)
	scope.addIndex(true, indexName, columns...)
	return scope.db
}

// RemoveIndex remove index with name
func (s *DB) RemoveIndex(indexName string) *DB {
	scope := s.NewScope(s.Val)
	scope.removeIndex(indexName)
	return scope.db
}

// AddForeignKey Add foreign key to the given scope, e.g:
//     db.Model(&User{}).AddForeignKey("city_id", "cities(id)", "RESTRICT", "RESTRICT")
func (s *DB) AddForeignKey(field string, dest string, onDelete string, onUpdate string) *DB {
	scope := s.NewScope(s.Val)
	scope.addForeignKey(field, dest, onDelete, onUpdate)
	return scope.db
}

// RemoveForeignKey Remove foreign key from the given scope, e.g:
//     db.Model(&User{}).RemoveForeignKey("city_id", "cities(id)")
func (s *DB) RemoveForeignKey(field string, dest string) *DB {
	scope := s.clone().NewScope(s.Val)
	scope.removeForeignKey(field, dest)
	return scope.db
}
