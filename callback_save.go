package aorm

import (
	"reflect"
)

func beginTransactionCallback(scope *Scope) {
	scope.Begin()
}

func commitOrRollbackTransactionCallback(scope *Scope) {
	scope.CommitOrRollback()
}

func saveAssociationCheck(scope *Scope, field *Field) (autoUpdate bool, autoCreate bool, saveReference bool, r *Relationship) {
	if scope.changeableField(field) && !field.IsBlank && !field.IsIgnored && (!field.IsChild || field.Relationship.Kind == "has_many") {
		if r = field.Relationship; r != nil {
			autoUpdate, autoCreate, saveReference = true, true, true

			if value, ok := scope.Get("aorm:save_associations"); ok {
				autoUpdate = checkTruth(value)
				autoCreate = autoUpdate
			} else if value, ok := field.TagSettings["SAVE_ASSOCIATIONS"]; ok {
				autoUpdate = checkTruth(value)
				autoCreate = autoUpdate
			}

			if value, ok := scope.Get("aorm:association_autoupdate"); ok {
				autoUpdate = checkTruth(value)
			} else if value, ok := field.TagSettings["ASSOCIATION_AUTOUPDATE"]; ok {
				autoUpdate = checkTruth(value)
			}

			if value, ok := scope.Get("aorm:association_autocreate"); ok {
				autoCreate = checkTruth(value)
			} else if value, ok := field.TagSettings["ASSOCIATION_AUTOCREATE"]; ok {
				autoCreate = checkTruth(value)
			}

			if value, ok := scope.Get("aorm:association_save_reference"); ok {
				saveReference = checkTruth(value)
			} else if value, ok := field.TagSettings["ASSOCIATION_SAVE_REFERENCE"]; ok {
				saveReference = checkTruth(value)
			}
		}
	}

	return
}

func saveBeforeAssociationsCallback(scope *Scope) {
	for _, field := range scope.Instance().RelatedFields() {
		autoUpdate, autoCreate, saveReference, relationship := saveAssociationCheck(scope, field)

		if relationship != nil && relationship.Kind == "belongs_to" {
			// skip ptr fields
			if field.Struct.Type.Kind() == reflect.Ptr {
				continue
			}

			fieldValue := field.Field.Addr().Interface()

			if ZeroIdOf(fieldValue) {
				if autoCreate {
					scope.Err(scope.NewDB().Save(fieldValue).Error)
				}
			} else if autoUpdate {
				scope.Err(scope.NewDB().Save(fieldValue).Error)
			}

			if saveReference {
				if len(relationship.ForeignFieldNames) != 0 {
					// set value's foreign key
					for idx, fieldName := range relationship.ForeignFieldNames {
						associationForeignName := relationship.AssociationForeignDBNames[idx]
						if foreignField, ok := scope.New(fieldValue).FieldByName(associationForeignName); ok {
							scope.Err(scope.SetColumn(fieldName, foreignField.Field.Interface()))
						}
					}
				}
			}
		}
	}
}

func saveAfterAssociationsCallback(scope *Scope) {
	for _, field := range scope.Instance().RelatedFields() {
		autoUpdate, autoCreate, saveReference, relationship := saveAssociationCheck(scope, field)

		if relationship != nil && (relationship.Model.Parent == nil || relationship.Kind == "has_many") && (relationship.Kind == "has_one" || relationship.Kind == "has_many" || relationship.Kind == "many_to_many") {
			value := field.Field

			switch value.Kind() {
			case reflect.Slice:
				for i := 0; i < value.Len(); i++ {
					newDB := scope.NewDB().ModelStruct(relationship.Model)
					elem := value.Index(i).Addr().Interface()
					newScope := newDB.NewModelScope(field.Model, elem)

					if saveReference {
						if relationship.JoinTableHandler == nil && len(relationship.ForeignFieldNames) != 0 {
							if ID := relationship.Model.GetID(elem); !ID.IsZero() {
								relationship.SetRelatedID(elem, ID)
							}
							for idx, fieldName := range relationship.ForeignFieldNames {
								associationForeignName := relationship.AssociationForeignFieldNames[idx]
								if f, ok := scope.instance.FieldsMap[associationForeignName]; ok {
									scope.Err(newScope.SetColumn(fieldName, f.Field.Interface()))
								}
							}
						}

						if relationship.PolymorphicType != "" {
							scope.Err(newScope.SetColumn(relationship.PolymorphicType, relationship.PolymorphicValue(scope.db.Context, scope.db.singularTable)))
						}
					}

					if newScope.PrimaryKeyZero() {
						if autoCreate {
							scope.Err(newDB.Save(elem).Error)
						}
					} else if autoUpdate {
						switch relationship.Kind {
						case "many_to_many":
							if value, ok := scope.Get("aorm:association_autoupdate:many_to_many"); ok {
								if checkTruth(value) {
									if scope.Err(newDB.Save(elem).Error) != nil {
										return
									}
								}
							}
						default:
							if scope.Err(newDB.Save(elem).Error) != nil {
								return
							}
						}
					}

					if !ZeroIdOf(newScope.Value) && saveReference && relationship.Kind != "many_to_many" {
						if joinTableHandler := relationship.JoinTableHandler; joinTableHandler != nil {
							scope.Err(joinTableHandler.Add(joinTableHandler, newDB, scope.Value, newScope.Value))
						}
					}
				}

				if relationship.Kind == "many_to_many" {
					assoc := &Association{scope: scope, column: field.Name, field: field}
					scope.Err(assoc.Replace(field.Field.Interface()).Error())
				}
			default:
				elem := value.Addr().Interface()
				newScope := scope.New(elem)

				if saveReference {
					if len(relationship.ForeignFieldNames) != 0 {
						for idx, fieldName := range relationship.ForeignFieldNames {
							associationForeignName := relationship.AssociationForeignDBNames[idx]
							if f, ok := scope.FieldByName(associationForeignName); ok {
								scope.Err(newScope.SetColumn(fieldName, f.Field.Interface()))
							}
						}
					}

					if relationship.PolymorphicType != "" {
						scope.Err(newScope.SetColumn(relationship.PolymorphicType, relationship.PolymorphicValue(scope.db.Context, scope.db.singularTable)))
					}
				}

				if newScope.PrimaryKeyZero() {
					if autoCreate {
						scope.Err(scope.NewDB().Save(elem).Error)
					}
				} else if autoUpdate {
					scope.Err(scope.NewDB().Save(elem).Error)
				}
			}
		}
	}
}
