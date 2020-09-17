package aorm

import (
	"fmt"
	"strings"
)

func (scope *Scope) createJoinTable(field *StructField) {
	if relationship := field.Relationship; relationship != nil && relationship.JoinTableHandler != nil {
		joinTableHandler := relationship.JoinTableHandler
		joinTable := joinTableHandler.Table(scope.db)
		if !scope.Dialect().HasTable(joinTable) {
			toStruct := StructOf(field.Struct.Type)

			var sqlTypes, primaryKeys []string
			for idx, fieldName := range relationship.ForeignFieldNames {
				if field, ok := scope.modelStruct.FieldsByName[fieldName]; ok {
					foreignKeyStruct := field.Clone()
					foreignKeyStruct.IsPrimaryKey = false
					foreignKeyStruct.TagSettings["IS_JOINTABLE_FOREIGNKEY"] = "true"
					delete(foreignKeyStruct.TagSettings, "AUTO_INCREMENT")
					sqlTypes = append(sqlTypes, scope.Quote(relationship.ForeignDBNames[idx])+" "+scope.Dialect().DataTypeOf(foreignKeyStruct.Structure()))
					primaryKeys = append(primaryKeys, scope.Quote(relationship.ForeignDBNames[idx]))
				}
			}

			for idx, fieldName := range relationship.AssociationForeignFieldNames {
				if field, ok := toStruct.FieldsByName[fieldName]; ok {
					foreignKeyStruct := field.Clone()
					foreignKeyStruct.IsPrimaryKey = false
					foreignKeyStruct.TagSettings["IS_JOINTABLE_FOREIGNKEY"] = "true"
					delete(foreignKeyStruct.TagSettings, "AUTO_INCREMENT")
					sqlTypes = append(sqlTypes, scope.Quote(relationship.AssociationForeignDBNames[idx])+" "+scope.Dialect().DataTypeOf(foreignKeyStruct.Structure()))
					primaryKeys = append(primaryKeys, scope.Quote(relationship.AssociationForeignDBNames[idx]))
				}
			}
			ddl := fmt.Sprintf("CREATE TABLE %v (%v, PRIMARY KEY (%v))%s",
				scope.Quote(joinTable), strings.Join(sqlTypes, ","),
				strings.Join(primaryKeys, ","),
				scope.getTableOptions())

			// TODO: implements auditor
			scope.Err(scope.NewDB().Table(joinTable).Exec(ddl).Error)
		}
		scope.NewDB().Table(joinTable).AutoMigrate(joinTableHandler)
	}
}

func (scope *Scope) createTable() *Scope {
	var tags []string
	var primaryKeys []string
	var primaryKeyInColumnType = false
	Struct := scope.Struct()
	for _, field := range Struct.Fields {
		if field.IsNormal {
			sqlTag := scope.Dialect().DataTypeOf(field.Structure())

			// Check if the primary key constraint was specified as
			// part of the column type. If so, we can only support
			// one column as the primary key.
			if strings.Contains(strings.ToLower(sqlTag), "primary key") {
				primaryKeyInColumnType = true
			}

			tags = append(tags, scope.Quote(field.DBName)+" "+sqlTag)
		}

		if field.IsPrimaryKey {
			primaryKeys = append(primaryKeys, scope.Quote(field.DBName))
		}
		scope.createJoinTable(field)
	}

	var primaryKeyStr string
	if len(primaryKeys) > 0 && !primaryKeyInColumnType {
		primaryKeyStr = fmt.Sprintf(", PRIMARY KEY (%v)", strings.Join(primaryKeys, ","))
	}

	scope.Query.Query = fmt.Sprintf("CREATE TABLE %v (%v %v)%s", scope.QuotedTableName(), strings.Join(tags, ","), primaryKeyStr, scope.getTableOptions())
	Struct.TypeCallbacks.TypeRegistrator.Call("CreateTable", Before, scope, nil)
	if scope.HasError() {
		return scope
	}
	scope.Raw(scope.Query.Query).Exec()
	if scope.HasError() {
		return scope
	}
	Struct.TypeCallbacks.TypeRegistrator.Call("CreateTable", After, scope, nil)
	if scope.HasError() {
		return scope
	}
	scope.Query.Query = ""

	scope.autoIndex()
	scope.autoForeignKeys()
	scope.createChildrenTables()
	return scope
}

func (scope *Scope) dropTable() *Scope {
	scope.Raw(fmt.Sprintf("DROP TABLE %v%s", scope.QuotedTableName(), scope.getTableOptions())).Exec()
	return scope
}

func (scope *Scope) modifyColumn(column string, typ string) {
	scope.db.AddError(scope.Dialect().ModifyColumn(scope.QuotedTableName(), scope.Quote(column), typ))
}

func (scope *Scope) dropColumn(column string) {
	scope.Raw(fmt.Sprintf("ALTER TABLE %v DROP COLUMN %v", scope.QuotedTableName(), scope.Quote(column))).Exec()
}

func (scope *Scope) addIndex(unique bool, indexName string, column ...string) {
	if scope.Dialect().HasIndex(scope.TableName(), indexName) {
		return
	}

	var columns []string
	for _, name := range column {
		columns = append(columns, scope.quoteIfPossible(name))
	}

	sqlCreate := "CREATE INDEX"
	if unique {
		sqlCreate = "CREATE UNIQUE INDEX"
	}

	scope.Raw(fmt.Sprintf("%s %v ON %v(%v) %v", sqlCreate, indexName, scope.QuotedTableName(), strings.Join(columns, ", "), scope.whereSQL())).Exec()
}

func (scope *Scope) autoForeignKeys() *DB {
	ms := scope.modelStruct
	scope.db.migrator.PostHandler(func(db *DB) (err error) {
		db = db.ModelStruct(ms)
		scope := db.NewModelScope(ms, ms.Value)

		for _, fk := range ms.ForeignKeys {
			def := fk.Definition(scope)
			if err = def.Create(db); err != nil {
				return
			}
		}
		return
	})
	return scope.db
}

func (scope *Scope) addForeignKey(field string, dest string, onDelete string, onUpdate string) {
	// Compatible with old generated key
	keyName := scope.Dialect().BuildKeyName(scope.TableName(), field, dest, "foreign")

	if scope.Dialect().HasForeignKey(scope.TableName(), keyName) {
		return
	}
	var query = `ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s ON DELETE %s ON UPDATE %s;`
	scope.Raw(fmt.Sprintf(query, scope.QuotedTableName(), scope.quoteIfPossible(keyName), scope.quoteIfPossible(field), dest, onDelete, onUpdate)).Exec()
}

func (scope *Scope) removeForeignKey(field string, dest string) {
	keyName := scope.Dialect().BuildKeyName(scope.TableName(), field, dest, "foreign")
	if !scope.Dialect().HasForeignKey(scope.TableName(), keyName) {
		return
	}
	var mysql mysql
	var query string
	if scope.Dialect().GetName() == mysql.GetName() {
		query = `ALTER TABLE %s DROP FOREIGN KEY %s;`
	} else {
		query = `ALTER TABLE %s DROP CONSTRAINT %s;`
	}
	scope.Raw(fmt.Sprintf(query, scope.QuotedTableName(), scope.quoteIfPossible(keyName))).Exec()
}

func (scope *Scope) removeIndex(indexName string) {
	scope.Dialect().RemoveIndex(scope.TableName(), indexName)
}

func (scope *Scope) autoMigrate(parentScope *Scope) *Scope {
	tableName := scope.TableName()
	quotedTableName := scope.QuotedTableName()
	scope.modelStruct.TypeCallbacks.TypeRegistrator.Call("Migrate", Before, scope, parentScope)

	if !scope.Dialect().HasTable(tableName) {
		scope.createTable()

		if !scope.HasError() {
			scope.modelStruct.TypeCallbacks.TypeRegistrator.Call("Migrate", After, scope, parentScope)
		}
	} else {
		for _, field := range scope.Struct().Fields {
			if field.IsNormal && !field.IsReadOnly {
				if !scope.Dialect().HasColumn(tableName, field.DBName) {
					sqlTag := scope.Dialect().DataTypeOf(field.Structure())
					scope.Raw(fmt.Sprintf("ALTER TABLE %v ADD %v %v;", quotedTableName, scope.Quote(field.DBName), sqlTag)).Exec()
					if scope.HasError() {
						return scope
					}
				}
			}
			scope.createJoinTable(field)
			if scope.HasError() {
				return scope
			}
		}
		scope.autoIndex()
		scope.autoForeignKeys()
		if !scope.HasError() {
			scope.modelStruct.TypeCallbacks.TypeRegistrator.Call("Migrate", After, scope, parentScope)
		}
		scope.createChildrenTables()
	}
	return scope
}

func (scope *Scope) createChildrenTables() {
	for _, child := range scope.modelStruct.Children {
		childScope := scope.db.NewScope(child.Value)
		childScope.modelStruct = child
		childScope.autoMigrate(scope)
	}
	for _, child := range scope.modelStruct.HasManyChildren {
		childScope := scope.db.NewScope(child.Value)
		childScope.modelStruct = child
		childScope.autoMigrate(scope)
	}
}

func (scope *Scope) autoIndex() *Scope {
	tableName := scope.TableName()
	for _, ix := range scope.modelStruct.Indexes {
		name, sql := ix.SqlCreate(scope.db.dialect, tableName)
		if !scope.Dialect().HasIndex(tableName, name) {
			scope.db.AddError(scope.Raw(sql).Exec().Error())
		}
	}

	for _, ix := range scope.modelStruct.UniqueIndexes {
		name, sql := ix.SqlCreate(scope.db.dialect, tableName)
		if !scope.Dialect().HasIndex(tableName, name) {
			scope.db.AddError(scope.Raw(sql).Exec().Error())
		}
	}

	return scope
}
