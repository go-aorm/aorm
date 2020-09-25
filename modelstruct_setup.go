package aorm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/pkg/errors"
)

func (this *ModelStruct) setup(pth []string, embedded bool, from *ModelStruct) (err error) {
	if !embedded {
		defer func() {
			if err == nil && this.Parent == nil {
				this.updateChildren()
			}
			if err == nil {
				err = this.setupIndexes()
			}
			if cb, ok := this.Value.(AfterStructSetuper); ok {
				cb.AormAfterStructSetup(this)
			}
		}()
	}

	reflectType := this.Type

	var hasManyField = func(field *StructField) {
		if err != nil {
			return
		}
		var (
			relationship = &Relationship{
				Field: field,
			}
			foreignKeys            []string
			associationForeignKeys []string
			elemType               = field.Struct.Type
			foreignStruct          *ModelStruct
		)
		for elemType.Kind() == reflect.Slice || elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}
		if foreignStruct, err = this.storage.getOrNew(field.Struct.Type, nil, false, this); err != nil {
			err = errors.Wrapf(err, "model struct of field %q", field.Name)
			return
		}
		if elemType.Name() == "" || foreignStruct.Tags.Flag("INLINE") || field.TagSettings.Flag("INLINE") {
			err = this.addChildSliceField(append(pth, this.Name), field, foreignStruct)
			return
		}

		field.Model = foreignStruct
		relationship.Model = foreignStruct
		relationship.AssociationModel = field.BaseModel
		toFields := foreignStruct.newFields()

		if fieldName := field.TagSettings["FIELD"]; fieldName != "" {
			relationship.FieldName = fieldName
		}
		if foreignKey := field.TagSettings["FOREIGNKEY"]; foreignKey != "" {
			if fkc := field.TagSettings["FKC"]; fkc != "" && fkc != "FKC" {
				panic(fmt.Errorf("%s#%s: foreign key tag duplication", this.Fqn(), field.Name))
			}
			field.TagSettings["FKC"] = "{field:{" + strings.ReplaceAll(foreignKey, ",", ";") + "}}"
			delete(field.TagSettings, "FOREIGNKEY")
		}
		if fkc := field.TagSettings["FKC"]; fkc != "" {
			var fk = &ForeignKey{Field: field}
			if fkcTags := field.TagSettings.GetTags("FKC"); fkcTags != nil {
				fk.OnDelete = fkcTags["DELETE"]
				fk.OnUpdate = fkcTags["UPDATE"]
				fk.Name = fkcTags["NAME"]
				if fkcTags.Flag("CASCADE") {
					fk.OnUpdate = "CASCADE"
					fk.OnDelete = "CASCADE"
				}

				if name := fkcTags.GetString("FIELD"); name != "" {
					foreignKeys = strings.Split(name, ";")
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
			}
			if fk.OnDelete == "" {
				fk.OnDelete = "SET NULL"
			}
			if fk.OnUpdate == "" {
				fk.OnUpdate = "CASCADE"
			}
			foreignStruct.ForeignKeys = append(foreignStruct.ForeignKeys, fk)
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
				var (
					prefix  string
					tbNamer JoinTableHandlerNamer
				)

				if many2many == "M2M" {
					prefix, _ = JoinNameOfString(
						this.TableName(context.Background(), true),
						field.DBName,
					)
					tbNamer = func(singular bool, src, dst *JoinTableSource) string {
						var _, tableName = JoinNameOfString(
							src.ModelStruct.TableName(context.Background(), singular),
							field.DBName,
						)
						return tableName
					}
				} else {
					tbNamer = DefaultM2MNamer(field)
				}

				{ // Foreign Keys for Source
					joinTableDBNames := []string{}

					if foreignKey := field.TagSettings["JOINTABLE_FOREIGNKEY"]; foreignKey != "" {
						joinTableDBNames = strings.Split(foreignKey, ",")
					}

					// if no foreign keys defined with tag
					if len(foreignKeys) == 0 {
						for _, field := range this.PrimaryFields {
							foreignKeys = append(foreignKeys, field.Name)
						}
					}

					for idx, foreignKey := range foreignKeys {
						if foreignField := this.FieldsByName[foreignKey]; foreignField != nil {
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
							associationForeignKeys = append(associationForeignKeys, field.Name)
						}
					}

					for idx, name := range associationForeignKeys {
						if field, ok := toFields.FieldsMap[name]; ok {
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
				joinTableHandler.Setup(relationship, tbNamer, this, foreignStruct)
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
	}

	// Get all fields
	for _, fieldStruct := range getTypeFields(reflectType) {
		if fieldStruct.Name == "_" {
			continue
		}

		field := &StructField{
			Struct:      fieldStruct.StructField,
			Name:        fieldStruct.Name,
			Names:       []string{fieldStruct.Name},
			Tag:         fieldStruct.Tag,
			TagSettings: parseFieldTagSetting(fieldStruct.StructField),
			StructIndex: fieldStruct.Index,
			Data:        map[interface{}]interface{}{},
			BaseModel:   this,
		}

		// is ignored field
		if _, ok := field.TagSettings["-"]; ok {
			field.IsIgnored = true
		} else {
			if field.Name == "ID" || field.TagSettings.Flag("PRIMARY_KEY") {
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

			indType := fieldStruct.Type
			for indType.Kind() == reflect.Ptr {
				indType = indType.Elem()
			}

			fieldValue := reflect.New(indType).Interface()

			if fieldAssigner, ok := fieldValue.(FieldAssigner); ok {
				field.Assigner = fieldAssigner.AormAssigner()
			} else if this.storage.GetAssigner != nil && field.Assigner == nil {
				field.Assigner = this.storage.GetAssigner(indType)
			}
			if selector, is := fieldValue.(FieldSelector); is {
				field.Selector = selector
			} else if sel := field.TagSettings["SELECT"]; sel != "" {
				field.Selector = NewFieldSelector(func(field *StructField, scope *Scope, expr string) Query {
					query := IQ(field.TagSettings["SELECT"]).WhereClause(scope)
					if field.IsReadOnly {
						if field.SelectWraper != nil {
							query.Query = "(" + query.Query + ")"
							return *field.SelectWraper.SelectWrap(field, scope, &query)
						}
					}
					return query
				})
			} else if _, ok := reflectType.MethodByName("AormFieldSelect" + field.Name); ok {
				field.Selector = NewFieldSelector(func(field *StructField, scope *Scope, expr string) (query Query) {
					query = scope.IndirectValue().MethodByName("AormFieldSelect" + field.Name).Call([]reflect.Value{
						reflect.ValueOf(field),
						reflect.ValueOf(scope),
						reflect.ValueOf(expr),
					})[0].Interface().(Query)

					if field.IsReadOnly {
						if field.SelectWraper != nil {
							query.Query = "(" + query.Query + ")"
							return *field.SelectWraper.SelectWrap(field, scope, &query)
						}
					}
					return query
				})
			}

			if field.IsReadOnly = field.TagSettings.Flag("RO"); !field.IsReadOnly {
				if ro, ok := fieldValue.(FieldReadOnlier); ok {
					field.IsReadOnly = ro.AormFieldReadOnly()
				}
			}

			if _, isScanner := fieldValue.(sql.Scanner); isScanner && !fieldStruct.Anonymous {
				// is scanner
				field.IsScanner, field.IsNormal = true, !field.IsReadOnly

				if indType.Kind() == reflect.Struct {
					for i := 0; i < indType.NumField(); i++ {
						for key, value := range parseFieldTagSetting(indType.Field(i)) {
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
				if subModel, err = this.storage.getOrNew(fieldValue, []string{this.Name}, true, this); err != nil {
					return errors.Wrapf(err, "model struct for field %q", field.Name)
				}

				if fieldStruct.Anonymous {
					for _, fk := range subModel.ForeignKeys {
						this.ForeignKeys = append(this.ForeignKeys, fk.Clone())
					}
				}

				// is embedded struct
				for _, subFieldOriginal := range subModel.Fields {
					subField := subFieldOriginal.Clone()
					subField.BaseModel = this
					if subField.Relationship != nil {
						subField.Relationship = subField.Relationship.Copy()
						subField.Relationship.Model = this
						if subField.Relationship.Field == subFieldOriginal {
							subField.Relationship.Field = subField
						}
					}

					subField.Names = append([]string{fieldStruct.Name}, subField.Names...)
					if prefix, ok := field.TagSettings["EMBEDDED_PREFIX"]; ok {
						subField.DBName = prefix + "_" + subField.DBName
					}

					if fieldStruct.Anonymous {
						subField.StructIndex = append(fieldStruct.Index, subField.StructIndex...)
						if subField.IsChild {
							childModel := subField.Model.Clone()
							childModel.ParentField = subField
							childModel.PluralTableName = ""
							childModel.SingularTableName = ""
							childModel.Parent = this

							subField.Relationship.Model = this
							subField.Relationship.AssociationModel = childModel

							newParentFk := childModel.ParentForeignKey.Clone()
							newParentFk.Field = subField
							childModel.ForeignKeys = append([]*ForeignKey{}, childModel.ForeignKeys...)

							for i, fk := range childModel.ForeignKeys {
								if fk == childModel.ParentForeignKey {
									childModel.ForeignKeys[i] = newParentFk
								}
							}
							childModel.ParentForeignKey = newParentFk

							subField.Model = childModel

							this.ChildrenByName[subField.Name] = childModel
							this.Children = append(this.Children, childModel)
							this.Fields = append(this.Fields, subField)
							continue
						}
					}

					if subField.IsPrimaryKey {
						if subField.TagSettings.Flag("PRIMARY_KEY") {
							this.PrimaryFields = append(this.PrimaryFields, subField)
						} else {
							subField.IsPrimaryKey = false
						}
					}

					this.Fields = append(this.Fields, subField)
					if subField.Relationship != nil {
						defer func(field *StructField) {
							if field.Relationship.JoinTableHandler != nil {
								hasManyField(field)
							} else {
								this.RelatedFields = append(this.RelatedFields, field)
							}
						}(subField)
					}
				}
				continue
			} else {
				// build relationships
				switch indType.Kind() {
				case reflect.Slice:
					elemTyp, _, _ := StructTypeOf(indType.Elem())
					if elemTyp == nil {
						field.IsNormal = true
					} else {
						defer hasManyField(field)
					}
				case reflect.Struct:
					defer func(field *StructField) {
						if err != nil {
							return
						}
						var (
							// user has one profile, associationType is User, profile use UserID as foreign key
							// user belongs to profile, associationType is Profile, user use ProfileID as foreign key
							associationType           = reflectType.Name()
							relationship              = &Relationship{}
							tagForeignKeys            []string
							tagAssociationForeignKeys []string
							foreignStruct             *ModelStruct
						)

						if foreignStruct, err = this.storage.getOrNew(field.Struct.Type, nil, false, this); err != nil {
							err = errors.Wrapf(err, "model struct of field %q", field.Name)
							return
						}

						if indirectType(field.Struct.Type).Name() == "" && len(this.PrimaryFields) > 0 {
							err = this.addChildField(field, foreignStruct)
							return
						}

						if len(foreignStruct.PrimaryFields) == 0 {
							err = errors.Wrapf(errors.New("child model require primary key"), "child of field %q", field.Name)
							return
						} else if field.TagSettings["CHILD"] != "" || foreignStruct.Tags.Flag("CHILD") ||
							(reflect.PtrTo(indirectType(field.Struct.Type)).Implements(reflect.TypeOf((*CanChilder)(nil)).Elem()) &&
								reflect.New(indirectType(field.Struct.Type)).Interface().(CanChilder).AormCanChild()) {
							if err = this.addChildField(field, foreignStruct); err != nil {
								err = errors.Wrapf(err, "field [as child] %q", field.Name)
							}
							return
						}

						relationship.Model = field.BaseModel
						relationship.AssociationModel = foreignStruct
						field.Model = foreignStruct

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
				if callbackMethod := MethodByName(indType, callbackName); callbackMethod.valid {
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

		if value, ok := field.TagSettings["COLUMN"]; ok {
			field.TagSettings["DB_NAME"] = value
			delete(field.TagSettings, "COLUMN")
		}

		// Even it is ignored, also possible to decode db value into the field
		if value, ok := field.TagSettings["DB_NAME"]; ok {
			field.DBName = value
		} else {
			field.DBName = ToDBName(fieldStruct.Name)
		}

		if field.IsReadOnly {
			this.ReadOnlyFields = append(this.ReadOnlyFields, field)
		}

		this.Fields = append(this.Fields, field)
	}

	if len(this.PrimaryFields) == 0 {
		if field := getForeignField("id", this.Fields); field != nil {
			field.IsPrimaryKey = true
			this.PrimaryFields = append(this.PrimaryFields, field)
		}
	}

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

	if len(this.PrimaryFields) == 0 && this.Parent == nil {
		p := this.parentTemp
		for p != nil && len(p.PrimaryFields) == 0 {
			p = p.Parent
		}
		if p != nil {
			this.Parent = from
		}
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
	setting.Parse(StructTag(field.Tag), "sql", "gorm", "aorm")
	return setting
}
