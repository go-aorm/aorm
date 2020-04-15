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
					foreignKeyStruct := field.clone()
					foreignKeyStruct.IsPrimaryKey = false
					foreignKeyStruct.TagSettings["IS_JOINTABLE_FOREIGNKEY"] = "true"
					delete(foreignKeyStruct.TagSettings, "AUTO_INCREMENT")
					sqlTypes = append(sqlTypes, scope.Quote(relationship.ForeignDBNames[idx])+" "+scope.Dialect().DataTypeOf(foreignKeyStruct.Structure()))
					primaryKeys = append(primaryKeys, scope.Quote(relationship.ForeignDBNames[idx]))
				}
			}

			for idx, fieldName := range relationship.AssociationForeignFieldNames {
				if field, ok := toStruct.FieldsByName[fieldName]; ok {
					foreignKeyStruct := field.clone()
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
			scope.Err(scope.NewDB().Exec(ddl).Error)
		}
		scope.NewDB().Table(joinTable).AutoMigrate(joinTableHandler)
	}
}

func (scope *Scope) createTable() *Scope {
	var tags []string
	var primaryKeys []string
	var primaryKeyInColumnType = false
	for _, field := range scope.Struct().Fields {
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

	scope.Raw(fmt.Sprintf("CREATE TABLE %v (%v %v)%s", scope.QuotedTableName(), strings.Join(tags, ","), primaryKeyStr, scope.getTableOptions())).Exec()

	scope.autoIndex()
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

func (scope *Scope) autoMigrate() *Scope {
	tableName := scope.TableName()
	quotedTableName := scope.QuotedTableName()

	if !scope.Dialect().HasTable(tableName) {
		scope.createTable()
	} else {
		for _, field := range scope.Struct().Fields {
			if !scope.Dialect().HasColumn(tableName, field.DBName) {
				if field.IsNormal && !field.IsReadOnly {
					sqlTag := scope.Dialect().DataTypeOf(field.Structure())
					scope.Raw(fmt.Sprintf("ALTER TABLE %v ADD %v %v;", quotedTableName, scope.Quote(field.DBName), sqlTag)).Exec()
				}
			}
			scope.createJoinTable(field)
		}
		scope.autoIndex()
	}
	return scope
}

func (scope *Scope) autoIndex() *Scope {
	var indexes = map[string]*struct {
		columns []string
		where   []string
	}{}
	var uniqueIndexes = map[string]*struct {
		columns []string
		where   []string
	}{}

	fields := scope.GetStructFields()

	for _, field := range fields {
		if name, ok := field.TagSettings["INDEX"]; ok {
			name = strings.ReplaceAll(name, "TB", scope.TableName())
			names := strings.Split(name, ",")

			for _, name := range names {
				ix := &struct {
					columns []string
					where   []string
				}{}
				parts := strings.SplitN(name, "=", 2)
				if len(parts) == 2 {
					name = parts[0]
					ix.where = append(ix.where, strings.ReplaceAll(parts[1], "{}", field.DBName))
				}
				if name == "INDEX" || name == "" {
					name = scope.Dialect().BuildKeyName("idx", scope.TableName(), field.DBName)
				}
				ix.columns = append(ix.columns, field.DBName)
				if old, ok := indexes[name]; ok {
					old.columns = append(old.columns, ix.columns...)
					old.where = append(old.where, ix.where...)
				} else {
					indexes[name] = ix
				}
			}
		}

		if name, ok := field.TagSettings["UNIQUE_INDEX"]; ok {
			name = strings.ReplaceAll(name, "TB", scope.TableName())
			names := strings.Split(name, ",")

			for _, name := range names {
				ix := &struct {
					columns []string
					where   []string
				}{}
				parts := strings.SplitN(name, "=", 2)
				if len(parts) == 2 {
					name = parts[0]
					ix.where = append(ix.where, strings.ReplaceAll(parts[1], "{}", field.DBName))
				}
				if name == "UNIQUE_INDEX" || name == "" {
					name = scope.Dialect().BuildKeyName("uix", scope.TableName(), field.DBName)
				}
				ix.columns = append(ix.columns, field.DBName)
				if old, ok := uniqueIndexes[name]; ok {
					old.columns = append(old.columns, ix.columns...)
					old.where = append(old.where, ix.where...)
				} else {
					uniqueIndexes[name] = ix
				}
			}
		}
	}

	for name, ix := range indexes {
		s := scope.NewDB().Table(scope.TableName()).Model(scope.Value)
		if len(ix.where) > 0 {
			s = s.Where(strings.Join(ix.where, " AND "))
		}
		if db := s.AddIndex(name, ix.columns...); db.Error != nil {
			scope.db.AddError(db.Error)
		}
	}

	for name, ix := range uniqueIndexes {
		s := scope.NewDB().Table(scope.TableName()).Model(scope.Value)
		if len(ix.where) > 0 {
			s = s.Where(strings.Join(ix.where, " AND "))
		}
		if db := s.AddUniqueIndex(name, ix.columns...); db.Error != nil {
			scope.db.AddError(db.Error)
		}
	}

	return scope
}
