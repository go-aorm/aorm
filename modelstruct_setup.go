package aorm

import (
	"context"
	"database/sql"
	"go/ast"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func (this *ModelStruct) setup(embedded bool) (err error) {
	if !embedded {
		defer func() {
			if err == nil {
				err = this.setupIndexes()
			}
		}()
	}

	reflectType := this.Type

	// Get all fields
	for i := 0; i < reflectType.NumField(); i++ {
		if fieldStruct := reflectType.Field(i); ast.IsExported(fieldStruct.Name) {
			field := &StructField{
				Struct:      fieldStruct,
				Name:        fieldStruct.Name,
				Names:       []string{fieldStruct.Name},
				Tag:         fieldStruct.Tag,
				TagSettings: parseFieldTagSetting(fieldStruct),
				StructIndex: fieldStruct.Index,
				Data:        map[interface{}]interface{}{},
				BaseModel:   this,
			}

			// is ignored field
			if _, ok := field.TagSettings["-"]; ok {
				field.IsIgnored = true
			} else {
				if field.TagSettings.Flag("PRIMARY_KEY") {
					field.IsPrimaryKey = true
					this.PrimaryFields = append(this.PrimaryFields, field)
				}

				if field.TagSettings.Flag("DEFAULT") {
					field.HasDefaultValue = true
				}

				if field.TagSettings.Flag("SERIAL") && !field.IsPrimaryKey {
					field.TagSettings["AUTO_INCREMENT"] = "AUTO_INCREMENT"
				}

				if field.TagSettings.Flag("AUTO_INCREMENT") {
					field.TagSettings["SERIAL"] = "SERIAL"
					if !field.IsPrimaryKey {
						field.HasDefaultValue = true
					}
				}

				indirectType := fieldStruct.Type
				for indirectType.Kind() == reflect.Ptr {
					indirectType = indirectType.Elem()
				}

				fieldValue := reflect.New(indirectType).Interface()

				if fieldAssigner, ok := fieldValue.(FieldAssigner); ok {
					field.Assigner = fieldAssigner.AormAssigner()
				} else if this.storage.GetAssigner != nil && field.Assigner == nil {
					field.Assigner = this.storage.GetAssigner(indirectType)
				}
				if selector, is := fieldValue.(FieldSelector); is {
					field.Selector = selector
				} else if sel := field.TagSettings["SELECT"]; sel != "" {
					field.Selector = NewFieldSelector(func(field *StructField, scope *Scope, expr string) Query {
						query := IQ(field.TagSettings["SELECT"]).WhereClause(scope)
						if field.IsReadOnly {
							if field.SelectWraper != nil {
								return field.SelectWraper.SelectWrap(field, scope, "("+query.Query+")")
							}
						}
						return query
					})
				}

				field.IsReadOnly = field.TagSettings.Flag("RO")

				if _, isScanner := fieldValue.(sql.Scanner); isScanner {
					// is scanner
					field.IsScanner, field.IsNormal = true, !field.IsReadOnly

					if indirectType.Kind() == reflect.Struct {
						for i := 0; i < indirectType.NumField(); i++ {
							for key, value := range parseFieldTagSetting(indirectType.Field(i)) {
								if _, ok := field.TagSettings[key]; !ok {
									field.TagSettings[key] = value
								}
							}
						}
					}
				} else if _, isTime := fieldValue.(*time.Time); isTime {
					// is time
					field.IsNormal = true
				} else if field.TagSettings.Flag("EMBEDDED") || fieldStruct.Anonymous {
					var subModel *ModelStruct
					if subModel, err = this.storage.getOrNew(fieldValue, true); err != nil {
						return errors.Wrapf(err, "model struct for field %q", field.Name)
					}
					// is embedded struct
					for _, subField := range subModel.Fields {
						subField = subField.clone()
						subField.BaseModel = this

						subField.Names = append([]string{fieldStruct.Name}, subField.Names...)
						if prefix, ok := field.TagSettings["EMBEDDED_PREFIX"]; ok {
							subField.DBName = prefix + "_" + subField.DBName
						}

						if subField.IsPrimaryKey {
							if subField.TagSettings.Flag("PRIMARY_KEY") {
								this.PrimaryFields = append(this.PrimaryFields, subField)
							} else {
								subField.IsPrimaryKey = false
							}
						}

						if subField.Relationship != nil && subField.Relationship.JoinTableHandler != nil {
							if joinTableHandler, ok := subField.Relationship.JoinTableHandler.(*JoinTableHandler); ok {
								newJoinTableHandler := &JoinTableHandler{}
								newJoinTableHandler.Setup(subField.Relationship, joinTableHandler.tableName, this, joinTableHandler.destination.ModelStruct)
								subField.Relationship.JoinTableHandler = newJoinTableHandler
							}
						}

						if fieldStruct.Anonymous {
							subField.StructIndex = append(fieldStruct.Index, subField.StructIndex...)
						}

						this.Fields = append(this.Fields, subField)
					}
					continue
				} else {
					// build relationships
					switch indirectType.Kind() {
					case reflect.Slice:
						elemTyp, _, _ := StructTypeOf(indirectType.Elem())
						if elemTyp == nil {
							field.IsNormal = true
						} else {
							defer func(field *StructField) {
								if err != nil {
									return
								}
								var (
									relationship           = &Relationship{}
									foreignKeys            []string
									associationForeignKeys []string
									elemType               = field.Struct.Type
									foreignStruct          *ModelStruct
								)
								for elemType.Kind() == reflect.Slice || elemType.Kind() == reflect.Ptr {
									elemType = elemType.Elem()
								}
								if foreignStruct, err = this.storage.GetOrNew(field.Struct.Type); err != nil {
									err = errors.Wrapf(err, "model struct of field %q", field.Name)
									return
								}
								relationship.Model = field.BaseModel
								relationship.AssociationModel = foreignStruct
								toFields := foreignStruct.newFields()

								if fieldName := field.TagSettings["FIELD"]; fieldName != "" {
									relationship.FieldName = fieldName
								}
								if foreignKey := field.TagSettings["FOREIGNKEY"]; foreignKey != "" {
									foreignKeys = strings.Split(foreignKey, ",")
									if relationship.FieldName == "" {
										parts := strings.Split(strings.ToLower(foreignKeys[0]), "id")
										relFieldName := foreignKeys[0][0:len(parts[0])]
										if f, ok := elemType.FieldByName(relFieldName); ok {
											fType := f.Type
											if fType.Kind() == reflect.Ptr {
												fType = fType.Elem()
											}
											if fType == reflectType {
												relationship.FieldName = relFieldName
											}
										}
									}
								}
								if foreignKey := field.TagSettings["ASSOCIATION_FOREIGNKEY"]; foreignKey != "" {
									associationForeignKeys = strings.Split(foreignKey, ",")
								} else if foreignKey := field.TagSettings["ASSOCIATIONFOREIGNKEY"]; foreignKey != "" {
									associationForeignKeys = strings.Split(foreignKey, ",")
								}

								if elemType.Kind() == reflect.Struct {
									if many2many := field.TagSettings["MANY2MANY"]; many2many != "" {
										if many2many == "MANY2MANY" {
											many2many = "M2M"
										}
										field.TagSettings["M2M"] = many2many
										delete(field.TagSettings, "MANY2MANY")
									}
									if many2many := field.TagSettings["M2M"]; many2many != "" {
										relationship.Kind = "many_to_many"
										var prefix string

										if many2many == "M2M" {
											prefix, many2many = M2MNameOf(reflectType, elemType)
										}

										{ // Foreign Keys for Source
											joinTableDBNames := []string{}

											if foreignKey := field.TagSettings["JOINTABLE_FOREIGNKEY"]; foreignKey != "" {
												joinTableDBNames = strings.Split(foreignKey, ",")
											}

											// if no foreign keys defined with tag
											if len(foreignKeys) == 0 {
												for _, field := range this.PrimaryFields {
													foreignKeys = append(foreignKeys, field.DBName)
												}
											}

											for idx, foreignKey := range foreignKeys {
												if foreignField := getForeignField(foreignKey, this.Fields); foreignField != nil {
													// source foreign keys (db names)
													relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)

													// setup join table foreign keys for source
													if len(joinTableDBNames) > idx {
														// if defined join table's foreign key
														relationship.ForeignDBNames = append(relationship.ForeignDBNames, joinTableDBNames[idx])
													} else {
														defaultJointableForeignKey := ToDBName(strings.TrimPrefix(reflectType.Name(), prefix)) + "_" + foreignField.DBName
														relationship.ForeignDBNames = append(relationship.ForeignDBNames, defaultJointableForeignKey)
													}
												}
											}
										}

										{ // Foreign Keys for Association (destination)
											associationJoinTableDBNames := []string{}

											if foreignKey := field.TagSettings["ASSOCIATION_JOINTABLE_FOREIGNKEY"]; foreignKey != "" {
												associationJoinTableDBNames = strings.Split(foreignKey, ",")
											}

											// if no association foreign keys defined with tag
											if len(associationForeignKeys) == 0 {
												for _, field := range foreignStruct.PrimaryFields {
													associationForeignKeys = append(associationForeignKeys, field.DBName)
												}
											}

											for idx, name := range associationForeignKeys {
												if field, ok := toFields.FieldByName(name); ok {
													// association foreign keys (db names)
													relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, field.Name)

													// setup join table foreign keys for association
													if len(associationJoinTableDBNames) > idx {
														relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, associationJoinTableDBNames[idx])
													} else {
														// join table foreign keys for association
														joinTableDBName := ToDBName(strings.TrimPrefix(elemType.Name(), prefix)) + "_" + field.DBName
														relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, joinTableDBName)
													}
												}
											}
										}

										var joinTableHandler JoinTableHandler
										joinTableHandler.Setup(relationship, many2many, this, foreignStruct)
										relationship.JoinTableHandler = &joinTableHandler
										field.Relationship = relationship
									} else {
										// User has many comments, associationType is User, comment use UserID as foreign key
										var associationType = reflectType.Name()
										relationship.Kind = "has_many"

										if polymorphic := field.TagSettings["POLYMORPHIC"]; polymorphic != "" {
											// Dog has many toys, tag polymorphic is Owner, then associationType is Owner
											// Toy use OwnerID, OwnerType ('dogs') as foreign key
											if polymorphicType := getForeignField(polymorphic+"Type", foreignStruct.Fields); polymorphicType != nil {
												associationType = polymorphic
												relationship.PolymorphicType = polymorphicType.Name
												relationship.PolymorphicDBName = polymorphicType.DBName
												// if Dog has multiple set of toys set name of the set (instead of default 'dogs')
												if value, ok := field.TagSettings["POLYMORPHIC_VALUE"]; ok {
													relationship.PolymorphicValue = polymorphicValue(value)
												} else {
													relationship.PolymorphicValue = func(ctx context.Context, singular bool) string {
														return this.RealTableName(ctx, singular)
													}
												}
												polymorphicType.IsForeignKey = true
											}
										}

										// if no foreign keys defined with tag
										if len(foreignKeys) == 0 {
											// if no association foreign keys defined with tag
											if len(associationForeignKeys) == 0 {
												for _, field := range this.PrimaryFields {
													foreignKeys = append(foreignKeys, associationType+field.Name)
													associationForeignKeys = append(associationForeignKeys, field.Name)
												}
											} else {
												// generate foreign keys from defined association foreign keys
												for _, scopeFieldName := range associationForeignKeys {
													if foreignField := getForeignField(scopeFieldName, this.Fields); foreignField != nil {
														foreignKeys = append(foreignKeys, associationType+foreignField.Name)
														associationForeignKeys = append(associationForeignKeys, foreignField.Name)
													}
												}
											}
										} else {
											// generate association foreign keys from foreign keys
											if len(associationForeignKeys) == 0 {
												for _, foreignKey := range foreignKeys {
													if strings.HasPrefix(foreignKey, associationType) {
														associationForeignKey := strings.TrimPrefix(foreignKey, associationType)
														if foreignField := getForeignField(associationForeignKey, this.Fields); foreignField != nil {
															associationForeignKeys = append(associationForeignKeys, associationForeignKey)
														}
													}
												}
												if len(associationForeignKeys) == 0 && len(foreignKeys) == 1 {
													associationForeignKeys = []string{this.PrimaryFields[0].DBName}
												}
											} else if len(foreignKeys) != len(associationForeignKeys) {
												err = errors.New("invalid foreign keys, should have same length")
												return
											}
										}

										for idx, foreignKey := range foreignKeys {
											if foreignField := getForeignField(foreignKey, foreignStruct.Fields); foreignField != nil {
												if associationField := getForeignField(associationForeignKeys[idx], this.Fields); associationField != nil {
													// source foreign keys
													foreignField.IsForeignKey = true
													relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, associationField.Name)
													relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, associationField.DBName)

													// association foreign keys
													relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
													relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
												}
											}
										}

										if len(relationship.ForeignFieldNames) != 0 {
											field.Relationship = relationship
										}
									}
								}

								if field.Relationship != nil {
									this.RelatedFields = append(this.RelatedFields, field)
								}
							}(field)
						}
					case reflect.Struct:
						defer func(field *StructField) {
							var (
								// user has one profile, associationType is User, profile use UserID as foreign key
								// user belongs to profile, associationType is Profile, user use ProfileID as foreign key
								associationType           = reflectType.Name()
								relationship              = &Relationship{}
								tagForeignKeys            []string
								tagAssociationForeignKeys []string
								foreignStruct             *ModelStruct
							)
							if foreignStruct, err = this.storage.GetOrNew(field.Struct.Type); err != nil {
								err = errors.Wrapf(err, "model struct of field %q", field.Name)
								return
							}

							relationship.Model = field.BaseModel
							relationship.AssociationModel = foreignStruct

							if foreignKey := field.TagSettings["FOREIGNKEY"]; foreignKey != "" {
								tagForeignKeys = strings.Split(foreignKey, ",")
							}

							if foreignKey := field.TagSettings["ASSOCIATION_FOREIGNKEY"]; foreignKey != "" {
								tagAssociationForeignKeys = strings.Split(foreignKey, ",")
							} else if foreignKey := field.TagSettings["ASSOCIATIONFOREIGNKEY"]; foreignKey != "" {
								tagAssociationForeignKeys = strings.Split(foreignKey, ",")
							}

							if polymorphic := field.TagSettings["POLYMORPHIC"]; polymorphic != "" {
								// Cat has one toy, tag polymorphic is Owner, then associationType is Owner
								// Toy use OwnerID, OwnerType ('cats') as foreign key
								if polymorphicType := getForeignField(polymorphic+"Type", foreignStruct.Fields); polymorphicType != nil {
									associationType = polymorphic
									relationship.PolymorphicType = polymorphicType.Name
									relationship.PolymorphicDBName = polymorphicType.DBName
									// if Cat has several different types of toys set name for each (instead of default 'cats')
									if value, ok := field.TagSettings["POLYMORPHIC_VALUE"]; ok {
										relationship.PolymorphicValue = polymorphicValue(value)
									} else {
										relationship.PolymorphicValue = func(ctx context.Context, singular bool) string {
											return foreignStruct.RealTableName(ctx, singular)
										}
									}
									polymorphicType.IsForeignKey = true
								}
							}

							// Has One
							{
								var foreignKeys = tagForeignKeys
								var associationForeignKeys = tagAssociationForeignKeys
								// if no foreign keys defined with tag
								if len(foreignKeys) == 0 {
									// if no association foreign keys defined with tag
									if len(associationForeignKeys) == 0 {
										for _, primaryField := range this.PrimaryFields {
											foreignKeys = append(foreignKeys, associationType+primaryField.Name)
											associationForeignKeys = append(associationForeignKeys, primaryField.Name)
										}
									} else {
										// generate foreign keys form association foreign keys
										for _, associationForeignKey := range tagAssociationForeignKeys {
											if foreignField := getForeignField(associationForeignKey, this.Fields); foreignField != nil {
												foreignKeys = append(foreignKeys, associationType+foreignField.Name)
												associationForeignKeys = append(associationForeignKeys, foreignField.Name)
											}
										}
									}
								} else {
									// generate association foreign keys from foreign keys
									if len(associationForeignKeys) == 0 {
										for _, foreignKey := range foreignKeys {
											if strings.HasPrefix(foreignKey, associationType) {
												associationForeignKey := strings.TrimPrefix(foreignKey, associationType)
												if foreignField := getForeignField(associationForeignKey, this.Fields); foreignField != nil {
													associationForeignKeys = append(associationForeignKeys, associationForeignKey)
												}
											}
										}
										if len(associationForeignKeys) == 0 && len(foreignKeys) == 1 {
											associationForeignKeys = []string{foreignStruct.PrimaryFields[0].DBName}
										}
									} else if len(foreignKeys) != len(associationForeignKeys) {
										err = errors.New("invalid foreign keys, should have same length")
										return
									}
								}

								for idx, foreignKey := range foreignKeys {
									if foreignField := getForeignField(foreignKey, foreignStruct.Fields); foreignField != nil {
										if structField := getForeignField(associationForeignKeys[idx], this.Fields); structField != nil {
											foreignField.IsForeignKey = true
											// source foreign keys
											relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, structField.Name)
											relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, structField.DBName)

											// association foreign keys
											relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
											relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
										}
									}
								}
							}

							if len(relationship.ForeignFieldNames) != 0 {
								relationship.Kind = "has_one"
								field.Relationship = relationship
							} else {
								var (
									foreignKeys            = tagForeignKeys
									associationForeignKeys = tagAssociationForeignKeys
									toScope                *ModelStruct
								)
								if toScope, err = this.storage.GetOrNew(field.Struct.Type); err != nil {
									err = errors.Wrapf(err, "model struct of field %q", field.Name)
									return
								}

								if len(foreignKeys) == 0 {
									// generate foreign keys & association foreign keys
									if len(associationForeignKeys) == 0 {
										for _, primaryField := range toScope.PrimaryFields {
											foreignKeys = append(foreignKeys, field.Name+primaryField.Name)
											associationForeignKeys = append(associationForeignKeys, primaryField.Name)
										}
									} else {
										// generate foreign keys with association foreign keys
										for _, associationForeignKey := range associationForeignKeys {
											if foreignField := getForeignField(associationForeignKey, toScope.PrimaryFields); foreignField != nil {
												foreignKeys = append(foreignKeys, field.Name+foreignField.Name)
												associationForeignKeys = append(associationForeignKeys, foreignField.Name)
											}
										}
									}
								} else {
									// generate foreign keys & association foreign keys
									if len(associationForeignKeys) == 0 {
										for _, foreignKey := range foreignKeys {
											if strings.HasPrefix(foreignKey, field.Name) {
												associationForeignKey := strings.TrimPrefix(foreignKey, field.Name)
												if foreignField := getForeignField(associationForeignKey, toScope.PrimaryFields); foreignField != nil {
													associationForeignKeys = append(associationForeignKeys, associationForeignKey)
												}
											}
										}
										if len(associationForeignKeys) == 0 && len(foreignKeys) == 1 {
											associationForeignKeys = []string{toScope.PrimaryFields[0].DBName}
										}
									} else if len(foreignKeys) != len(associationForeignKeys) {
										err = errors.New("invalid foreign keys, should have same length")
										return
									}
								}

								for idx, foreignKey := range foreignKeys {
									if foreignField := getForeignField(foreignKey, this.Fields); foreignField != nil {
										if associationField := getForeignField(associationForeignKeys[idx], toScope.Fields); associationField != nil {
											foreignField.IsForeignKey = true

											// association foreign keys
											relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, associationField.Name)
											relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, associationField.DBName)

											// source foreign keys
											relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
											relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
										}
									}
								}

								if len(relationship.ForeignFieldNames) != 0 {
									relationship.Kind = "belongs_to"
									field.Relationship = relationship
								}
							}

							if field.Relationship != nil {
								this.RelatedFields = append(this.RelatedFields, field)
							}
						}(field)
					default:
						field.IsNormal = true
					}
				}

				// register method callbacks now for improve performance
				field.MethodCallbacks = make(map[string]StructFieldMethodCallback)

				for callbackName, caller := range StructFieldMethodCallbacks.Callbacks {
					if callbackMethod := MethodByName(indirectType, callbackName); callbackMethod.valid {
						field.MethodCallbacks[callbackName] = StructFieldMethodCallback{callbackMethod, caller}
					}
				}

				if field.Selector == nil {
					if selector, ok := field.Assigner.(FieldSelector); ok {
						field.Selector = selector
					}
				}

				if selectWraper, ok := fieldValue.(FieldSelectWraper); ok {
					field.SelectWraper = selectWraper
				} else if selectWraper, ok := field.Assigner.(FieldSelectWraper); ok {
					field.SelectWraper = selectWraper
				}
			}

			// Even it is ignored, also possible to decode db value into the field
			if value, ok := field.TagSettings["COLUMN"]; ok {
				field.DBName = value
			} else {
				field.DBName = ToDBName(fieldStruct.Name)
			}

			if field.IsReadOnly {
				this.ReadOnlyFields = append(this.ReadOnlyFields, field)
			}

			this.Fields = append(this.Fields, field)
		}
	}

	if len(this.PrimaryFields) == 0 {
		if field := getForeignField("id", this.Fields); field != nil {
			field.IsPrimaryKey = true
			this.PrimaryFields = append(this.PrimaryFields, field)
		}
	}

	this.FieldsByName = make(map[string]*StructField)
	this.DynamicFieldsByName = make(map[string]*StructField)

	for i, field := range this.Fields {
		field.Index = i
		this.FieldsByName[field.Name] = field
		if field.IsIgnored {
			this.IgnoredFieldsCount++
		}
		if field.IsReadOnly {
			this.DynamicFieldsByName[field.Name] = field
		}
	}

	if _, ok := this.FieldsByName[SoftDeleteFieldDeletedAt]; ok {
		this.softDelete = true
	}

	return
}

func polymorphicValue(value string) func(_ context.Context, singular bool) string {
	parts := strings.Split(value, ",")
	if len(parts) == 1 {
		return func(_ context.Context, singular bool) string {
			return value
		}
	} else {
		return func(_ context.Context, singular bool) string {
			if singular {
				return parts[0]
			}
			return parts[1]
		}
	}
}

func getForeignField(column string, fields []*StructField) *StructField {
	for _, field := range fields {
		if field.Name == column || field.DBName == column || field.DBName == ToDBName(column) {
			return field
		}
	}
	return nil
}

func parseFieldTagSetting(field reflect.StructField) (setting TagSetting) {
	setting = make(TagSetting)
	tags := field.Tag
	for _, str := range []string{tags.Get("sql"), tags.Get("gorm"), tags.Get("aorm")} {
		tags := strings.Split(str, ";")
		for _, value := range tags {
			v := strings.Split(value, ":")
			k := strings.TrimSpace(strings.ToUpper(v[0]))
			if len(v) >= 2 {
				setting[k] = strings.Join(v[1:], ":")
			} else {
				setting[k] = k
			}
		}
	}
	return
}
