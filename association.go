package aorm

import (
	"errors"
	"fmt"
	"reflect"
)

// Association Mode contains some helper methods to handle relationship things easily.
type Association struct {
	error  Errors
	scope  *Scope
	column string
	field  *Field
}

func (this *Association) Error() error {
	if len(this.error) == 0 {
		return nil
	}
	return this.error
}

func (this *Association) HasError() bool {
	return len(this.error) > 0
}

// Find find out all related associations
func (this *Association) Find(value interface{}) *Association {
	this.scope.related(value, this.column)
	return this.addErr(this.scope.db.Error)
}

// Append append new associations for many2many, has_many, replace current association for has_one, belongs_to
func (this *Association) Append(values ...interface{}) *Association {
	if this.error != nil {
		return this
	}

	if relationship := this.field.Relationship; relationship.Kind == "has_one" {
		return this.Replace(values...)
	}
	return this.saveAssociations(values...)
}

// Replace replace current associations with new one
func (this *Association) Replace(values ...interface{}) *Association {
	if this.error != nil {
		return this
	}

	var (
		relationship = this.field.Relationship
		scope        = this.scope
		field        = this.field.Field
		newDB        = scope.NewDB()
	)

	// Append new values
	this.field.Set(reflect.Zero(this.field.Field.Type()))
	this.saveAssociations(values...)

	// Belongs To
	if relationship.Kind == "belongs_to" {
		// Set foreign key to be null when clearing value (length equals 0)
		if len(values) == 0 {
			// Set foreign key to be nil
			var foreignKeyMap = map[string]interface{}{}
			for _, foreignKey := range relationship.ForeignDBNames {
				foreignKeyMap[foreignKey] = nil
			}
			this.addErr(newDB.Model(scope.Value).UpdateColumn(foreignKeyMap).Error)
		}
	} else {
		// Polymorphic Relations
		if relationship.PolymorphicDBName != "" {
			newDB = newDB.Where(fmt.Sprintf("%v = ?", scope.Quote(relationship.PolymorphicDBName)), relationship.PolymorphicValue)
		}

		// Delete Relations except new created
		if len(values) > 0 {
			var associationForeignFieldNames, associationForeignDBNames []string
			if relationship.Kind == "many_to_many" {
				// if many to many relations, get this fields name from this foreign keys
				instance := InstanceOf(reflect.New(field.Type()).Interface())
				for idx, fieldName := range relationship.AssociationForeignFieldNames {
					if field, ok := instance.FieldsMap[fieldName]; ok {
						associationForeignFieldNames = append(associationForeignFieldNames, field.Name)
						associationForeignDBNames = append(associationForeignDBNames, relationship.AssociationForeignDBNames[idx])
					}
				}
			} else {
				// If has one/many relations, use primary keys
				for _, field := range StructOf(field.Type()).PrimaryFieldsInstance(reflect.New(field.Type()).Interface()) {
					associationForeignFieldNames = append(associationForeignFieldNames, field.Name)
					associationForeignDBNames = append(associationForeignDBNames, field.DBName)
				}
			}

			newPrimaryKeys := scope.getColumnAsArray(associationForeignFieldNames, field.Interface())

			if len(newPrimaryKeys) > 0 {
				sql := fmt.Sprintf("%v NOT IN (%v)", toQueryCondition(scope, associationForeignDBNames), toQueryMarks(newPrimaryKeys))
				newDB = newDB.Where(sql, toQueryValues(newPrimaryKeys)...)
			}
		}

		if relationship.Kind == "many_to_many" {
			// if many to many relations, delete related relations from join table
			var sourceForeignFieldNames []string

			for _, fieldName := range relationship.ForeignFieldNames {
				if field, ok := scope.modelStruct.FieldsByName[fieldName]; ok {
					sourceForeignFieldNames = append(sourceForeignFieldNames, field.Name)
				}
			}

			if sourcePrimaryKeys := scope.getColumnAsArray(sourceForeignFieldNames, scope.Value); len(sourcePrimaryKeys) > 0 {
				newDB = newDB.Where(fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.ForeignDBNames), toQueryMarks(sourcePrimaryKeys)), toQueryValues(sourcePrimaryKeys)...)

				this.addErr(relationship.JoinTableHandler.Delete(relationship.JoinTableHandler, newDB))
			}
		} else if relationship.Kind == "has_one" || relationship.Kind == "has_many" {
			// has_one or has_many relations, set foreign key to be nil (TODO or delete them?)
			var foreignKeyMap = map[string]interface{}{}
			for idx, foreignKey := range relationship.ForeignDBNames {
				foreignKeyMap[foreignKey] = nil
				if field, ok := scope.FieldByName(relationship.AssociationForeignFieldNames[idx]); ok {
					newDB = newDB.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
				}
			}

			fieldValue := reflect.New(this.field.Field.Type()).Interface()
			this.addErr(newDB.Model(fieldValue).UpdateColumn(foreignKeyMap).Error)
		}
	}
	return this
}

// Delete remove relationship between source & passed arguments, but won'T delete those arguments
func (this *Association) Delete(values ...interface{}) *Association {
	if this.error != nil {
		return this
	}

	var (
		relationship = this.field.Relationship
		scope        = this.scope
		field        = this.field.Field
		newDB        = scope.NewDB()
	)

	if len(values) == 0 {
		return this
	}

	var deletingResourcePrimaryFieldNames, deletingResourcePrimaryDBNames []string
	for _, field := range scope.New(reflect.New(field.Type()).Interface()).PrimaryFields() {
		deletingResourcePrimaryFieldNames = append(deletingResourcePrimaryFieldNames, field.Name)
		deletingResourcePrimaryDBNames = append(deletingResourcePrimaryDBNames, field.DBName)
	}

	deletingPrimaryKeys := scope.getColumnAsArray(deletingResourcePrimaryFieldNames, values...)

	if relationship.Kind == "many_to_many" {
		// source value's foreign keys
		for idx, foreignKey := range relationship.ForeignDBNames {
			if field, ok := scope.FieldByName(relationship.ForeignFieldNames[idx]); ok {
				newDB = newDB.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
			}
		}

		// get this's foreign fields name
		var associationScope = scope.New(reflect.New(field.Type()).Interface())
		var associationForeignFieldNames []string
		for _, associationDBName := range relationship.AssociationForeignFieldNames {
			if field, ok := associationScope.FieldByName(associationDBName); ok {
				associationForeignFieldNames = append(associationForeignFieldNames, field.Name)
			}
		}

		// this value's foreign keys
		deletingPrimaryKeys := scope.getColumnAsArray(associationForeignFieldNames, values...)
		sql := fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.AssociationForeignDBNames), toQueryMarks(deletingPrimaryKeys))
		newDB = newDB.Where(sql, toQueryValues(deletingPrimaryKeys)...)

		this.addErr(relationship.JoinTableHandler.Delete(relationship.JoinTableHandler, newDB))
	} else {
		var foreignKeyMap = map[string]interface{}{}
		for _, foreignKey := range relationship.ForeignDBNames {
			foreignKeyMap[foreignKey] = nil
		}

		if relationship.Kind == "belongs_to" {
			// find with deleting relation's foreign keys
			primaryKeys := scope.getColumnAsArray(relationship.AssociationForeignFieldNames, values...)
			newDB = newDB.Where(
				fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.ForeignDBNames), toQueryMarks(primaryKeys)),
				toQueryValues(primaryKeys)...,
			)

			// set foreign key to be null if there are some records affected
			modelValue := reflect.New(scope.Struct().Type).Interface()
			if results := newDB.Model(modelValue).UpdateColumn(foreignKeyMap); results.Error == nil {
				if results.RowsAffected > 0 {
					scope.updatedAttrsWithValues(foreignKeyMap)
				}
			} else {
				this.addErr(results.Error)
			}
		} else if relationship.Kind == "has_one" || relationship.Kind == "has_many" {
			// find all relations
			primaryKeys := scope.getColumnAsArray(relationship.AssociationForeignFieldNames, scope.Value)
			newDB = newDB.Where(
				fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.ForeignDBNames), toQueryMarks(primaryKeys)),
				toQueryValues(primaryKeys)...,
			)

			// only include those deleting relations
			newDB = newDB.Where(
				fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, deletingResourcePrimaryDBNames), toQueryMarks(deletingPrimaryKeys)),
				toQueryValues(deletingPrimaryKeys)...,
			)

			// set matched relation's foreign key to be null
			fieldValue := reflect.New(this.field.Field.Type()).Interface()
			this.addErr(newDB.Model(fieldValue).UpdateColumn(foreignKeyMap).Error)
		}
	}

	// Remove deleted records from source's field
	if this.error == nil {
		if field.Kind() == reflect.Slice {
			leftValues := reflect.Zero(field.Type())

			for i := 0; i < field.Len(); i++ {
				reflectValue := field.Index(i)
				primaryKey := scope.getColumnAsArray(deletingResourcePrimaryFieldNames, reflectValue.Interface())[0]
				var isDeleted = false
				for _, pk := range deletingPrimaryKeys {
					if equalAsString(primaryKey, pk) {
						isDeleted = true
						break
					}
				}
				if !isDeleted {
					leftValues = reflect.Append(leftValues, reflectValue)
				}
			}

			this.field.Set(leftValues)
		} else if field.Kind() == reflect.Struct {
			primaryKey := scope.getColumnAsArray(deletingResourcePrimaryFieldNames, field.Interface())[0]
			for _, pk := range deletingPrimaryKeys {
				if equalAsString(primaryKey, pk) {
					this.field.Set(reflect.Zero(field.Type()))
					break
				}
			}
		}
	}

	return this
}

// Clear remove relationship between source & current associations, won'T delete those associations
func (this *Association) Clear() *Association {
	return this.Replace()
}

// Count return the count of current associations
func (this *Association) Count() int {
	var (
		count        = 0
		relationship = this.field.Relationship
		scope        = this.scope
		fieldValue   = this.field.Field.Interface()
		query        = scope.DB()
	)

	if relationship.Kind == "many_to_many" {
		query = relationship.JoinTableHandler.JoinWith(relationship.JoinTableHandler, query, scope.Value)
	} else if relationship.Kind == "has_many" || relationship.Kind == "has_one" {
		primaryKeys := scope.getColumnAsArray(relationship.AssociationForeignFieldNames, scope.Value)
		query = query.Where(
			fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.ForeignDBNames), toQueryMarks(primaryKeys)),
			toQueryValues(primaryKeys)...,
		)
	} else if relationship.Kind == "belongs_to" {
		primaryKeys := scope.getColumnAsArray(relationship.ForeignFieldNames, scope.Value)
		query = query.Where(
			fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relationship.AssociationForeignDBNames), toQueryMarks(primaryKeys)),
			toQueryValues(primaryKeys)...,
		)
	}

	if relationship.PolymorphicType != "" {
		query = query.Where(
			fmt.Sprintf("%v.%v = ?", scope.New(fieldValue).QuotedTableName(), scope.Quote(relationship.PolymorphicDBName)),
			relationship.PolymorphicValue(scope.db.Context, scope.db.singularTable),
		)
	}

	if err := query.Model(fieldValue).Count(&count).Error; err != nil {
		this.error.Add(err)
	}
	return count
}

// saveAssociations save passed values as associations
func (this *Association) saveAssociations(values ...interface{}) *Association {
	var (
		scope        = this.scope
		field        = this.field
		relationship = field.Relationship
	)

	saveAssociation := func(reflectValue reflect.Value) {
		// value has to been pointer
		if reflectValue.Kind() != reflect.Ptr {
			reflectPtr := reflect.New(reflectValue.Type())
			reflectPtr.Elem().Set(reflectValue)
			reflectValue = reflectPtr
		}

		// value has to been saved for many2many
		if relationship.Kind == "many_to_many" {
			if scope.New(reflectValue.Interface()).PrimaryKeyZero() {
				if err := scope.NewDB().Save(reflectValue.Interface()).Error; err != nil {
					this.addErr(err)
					return
				}
			}
		}

		// Assigner Instance
		var fieldType = field.Field.Type()
		var setFieldBackToValue, setSliceFieldBackToValue bool
		if reflectValue.Type().AssignableTo(fieldType) {
			field.Set(reflectValue)
		} else if reflectValue.Type().Elem().AssignableTo(fieldType) {
			// if field's type is struct, then need to set value back to argument after save
			setFieldBackToValue = true
			field.Set(reflectValue.Elem())
		} else if fieldType.Kind() == reflect.Slice {
			if reflectValue.Type().AssignableTo(fieldType.Elem()) {
				field.Set(reflect.Append(field.Field, reflectValue))
			} else if reflectValue.Type().Elem().AssignableTo(fieldType.Elem()) {
				// if field's type is slice of struct, then need to set value back to argument after save
				setSliceFieldBackToValue = true
				field.Set(reflect.Append(field.Field, reflectValue.Elem()))
			}
		}

		if relationship.Kind == "many_to_many" {
			err := relationship.JoinTableHandler.Add(relationship.JoinTableHandler, scope.NewDB(), scope.Value, reflectValue.Interface())
			if err != nil {
				this.addErr(err)
				return
			}
		} else {
			if err := scope.NewDB().Select(field.Name).Save(scope.Value).Error; err != nil {
				this.addErr(err)
				return
			}

			if setFieldBackToValue {
				reflectValue.Elem().Set(field.Field)
			} else if setSliceFieldBackToValue {
				reflectValue.Elem().Set(field.Field.Index(field.Field.Len() - 1))
			}
		}
	}

	for _, value := range values {
		reflectValue := reflect.ValueOf(value)
		indirectReflectValue := reflect.Indirect(reflectValue)
		if indirectReflectValue.Kind() == reflect.Struct {
			saveAssociation(reflectValue)
		} else if indirectReflectValue.Kind() == reflect.Slice {
			for i := 0; i < indirectReflectValue.Len(); i++ {
				saveAssociation(indirectReflectValue.Index(i))
			}
		} else {
			this.addErr(errors.New("invalid value type"))
		}
	}
	return this
}

func (this *Association) addErr(err error) *Association {
	this.error = this.error.Add(err)
	return this
}
