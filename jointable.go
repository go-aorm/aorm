package aorm

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// JoinTableForeignKey join table foreign key struct
type JoinTableForeignKey struct {
	DBName           string
	AssociationField *StructField
}

// JoinTableSource is a struct that contains model type and foreign keys
type JoinTableSource struct {
	ModelStruct *ModelStruct
	ForeignKeys []JoinTableForeignKey
}

// JoinTableHandler default join table handler
type JoinTableHandler struct {
	tableNameFunc func(singular bool) string
	source        JoinTableSource
	destination   JoinTableSource
}

func (this *JoinTableHandler) TableName(singular bool) string {
	return this.tableNameFunc(singular)
}

func (this *JoinTableHandler) Source() JoinTableSource {
	return this.source
}

func (this *JoinTableHandler) Destination() JoinTableSource {
	return this.destination
}

// SourceForeignKeys return source foreign keys
func (this *JoinTableHandler) SourceForeignKeys() []JoinTableForeignKey {
	return this.source.ForeignKeys
}

// DestinationForeignKeys return destination foreign keys
func (this *JoinTableHandler) DestinationForeignKeys() []JoinTableForeignKey {
	return this.destination.ForeignKeys
}

// Setup initialize a default join table handler
func (this *JoinTableHandler) Setup(relationship *Relationship, tableName func(singular bool) string, source, destination *ModelStruct) {
	this.tableNameFunc = tableName
	this.source = JoinTableSource{ModelStruct: source}
	this.source.ForeignKeys = []JoinTableForeignKey{}
	for idx, fieldName := range relationship.ForeignFieldNames {
		this.source.ForeignKeys = append(this.source.ForeignKeys, JoinTableForeignKey{
			DBName:           relationship.ForeignDBNames[idx],
			AssociationField: source.FieldsByName[fieldName],
		})
	}

	this.destination = JoinTableSource{ModelStruct: destination}
	this.destination.ForeignKeys = []JoinTableForeignKey{}
	for idx, fieldName := range relationship.AssociationForeignFieldNames {
		this.destination.ForeignKeys = append(this.destination.ForeignKeys, JoinTableForeignKey{
			DBName:           relationship.AssociationForeignDBNames[idx],
			AssociationField: destination.FieldsByName[fieldName],
		})
	}
}

// Table return join table's table name
func (this JoinTableHandler) Table(db *DB) string {
	return this.TableName(db.singularTable)
}

func (this JoinTableHandler) updateConditionMap(conditionMap map[string]interface{}, db *DB, joinTableSources []JoinTableSource, sources ...interface{}) {
	for _, source := range sources {
		instance := InstanceOf(source)
		modelType := instance.Struct.Type

		for _, joinTableSource := range joinTableSources {
			if joinTableSource.ModelStruct.Type == modelType {
				for _, foreignKey := range joinTableSource.ForeignKeys {
					if field, ok := instance.FieldsMap[foreignKey.AssociationField.Name]; ok {
						conditionMap[foreignKey.DBName] = field.Field.Interface()
					}
				}
				break
			}
		}
	}
}

// Add create relationship in join table for source and destination
func (this JoinTableHandler) Add(handler JoinTableHandlerInterface, db *DB, source interface{}, destination interface{}) (err error) {
	var (
		scope        = db.NewScope("")
		conditionMap = map[string]interface{}{}
	)

	// Update condition map for source
	this.updateConditionMap(conditionMap, db, []JoinTableSource{this.source}, source)

	// Update condition map for destination
	this.updateConditionMap(conditionMap, db, []JoinTableSource{this.destination}, destination)

	var assignColumns, binVars, conditions []string
	var values []interface{}
	for key, value := range conditionMap {
		assignColumns = append(assignColumns, scope.Quote(key))
		binVars = append(binVars, `?`)
		conditions = append(conditions, fmt.Sprintf("%v = ?", scope.Quote(key)))
		values = append(values, value)
	}

	for _, value := range values {
		values = append(values, value)
	}

	tableName := handler.Table(db)
	quotedTable := scope.Quote(tableName)
	sql := fmt.Sprintf(
		"INSERT INTO %v (%v) SELECT %v %v WHERE NOT EXISTS (SELECT * FROM %v WHERE %v)",
		quotedTable,
		strings.Join(assignColumns, ","),
		strings.Join(binVars, ","),
		scope.Dialect().SelectFromDummyTable(),
		quotedTable,
		strings.Join(conditions, " AND "),
	)
	if err = db.Table(tableName).Exec(sql, values...).Error; err != nil {
		return NewQueryError(err, Query{sql, values}, db.dialect.BindVar)
	}
	return
}

// Delete delete relationship in join table for sources
func (this JoinTableHandler) Delete(handler JoinTableHandlerInterface, db *DB, sources ...interface{}) error {
	var (
		scope        = db.NewScope(nil)
		conditions   []string
		values       []interface{}
		conditionMap = map[string]interface{}{}
	)

	this.updateConditionMap(conditionMap, db, []JoinTableSource{this.source, this.destination}, sources...)

	for key, value := range conditionMap {
		conditions = append(conditions, fmt.Sprintf("%v = ?", scope.Quote(key)))
		values = append(values, value)
	}

	return db.
		Table(handler.Table(db)).
		Where(strings.Join(conditions, " AND "), values...).
		Delete(reflect.New(handler.Destination().ModelStruct.Type).Interface()).Error
}

// JoinWith query with `Join` conditions
func (this JoinTableHandler) JoinWith(handler JoinTableHandlerInterface, db *DB, source interface{}) *DB {
	if indirectType(this.source.ModelStruct.Type) == indirectType(reflect.TypeOf(source)) {
		var (
			scope           = db.NewModelScope(this.source.ModelStruct, source)
			tableName       = handler.Table(db)
			quotedTableName = scope.Quote(tableName)
			joinConditions  []string
			values          []interface{}
			dialect         = db.dialect
		)
		destinationTableName := db.NewModelScope(this.destination.ModelStruct, this.destination.ModelStruct.Value).QuotedTableName()
		for _, foreignKey := range this.destination.ForeignKeys {
			joinConditions = append(joinConditions, fmt.Sprintf("%v.%v = %v.%v", quotedTableName, Quote(dialect, foreignKey.DBName), destinationTableName, Quote(dialect, foreignKey.AssociationField.DBName)))
		}

		var foreignDBNames []string
		var foreignFieldNames []string

		for _, foreignKey := range this.source.ForeignKeys {
			foreignDBNames = append(foreignDBNames, foreignKey.DBName)
			foreignFieldNames = append(foreignFieldNames, foreignKey.AssociationField.Name)
		}

		foreignFieldValues := scope.getColumnAsArray(foreignFieldNames, scope.Value)

		var condString string
		if len(foreignFieldValues) > 0 {
			var quotedForeignDBNames []string
			for _, dbName := range foreignDBNames {
				quotedForeignDBNames = append(quotedForeignDBNames, tableName+"."+dbName)
			}

			condString = fmt.Sprintf("%v IN (%v)", toQueryCondition(dialect, quotedForeignDBNames), toQueryMarks(foreignFieldValues))

			keys := scope.getColumnAsArray(foreignFieldNames, scope.Value)
			values = append(values, toQueryValues(keys))
		} else {
			condString = fmt.Sprintf("1 <> 1")
		}

		return db.Joins(fmt.Sprintf("INNER JOIN %v ON %v", quotedTableName, strings.Join(joinConditions, " AND "))).
			Where(condString, toQueryValues(foreignFieldValues)...)
	}

	db.Error = errors.New("wrong source type for join table handler")
	return db
}

func (this JoinTableHandler) Copy() JoinTableHandlerInterface {
	return &this
}
