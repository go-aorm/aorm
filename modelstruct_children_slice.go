package aorm

import (
	"fmt"
	"reflect"
	"strings"
)

func (this *ModelStruct) addChildSliceField(pth []string, field *StructField, ms *ModelStruct) (err error) {
	this.storage.ModelStructsMap.delete(ms.Type)
	pth = append(pth, field.Name)
	ms.Name = strings.Join(pth, "")
	ms.ParentField = field
	ms.HasManyChild = true
	ms.Parent = this
	if this.Root != nil {
		ms.Root = this.Root
	} else {
		ms.Root = this
	}

	if len(ms.PrimaryFields) == 0 {
		return fmt.Errorf("Model %s#%s does not have primary field", ms.Root.Type, field.Name)
	}

	this.HasManyChildrenByName[field.Name] = ms
	this.HasManyChildren = append(this.Children, ms)

	field.Model = ms

	primaryFields := this.PrimaryFields
	if this.Parent != nil {
		primaryFields = this.Root.PrimaryFields
	}

	belongsToRel := &Relationship{
		Model:            ms,
		AssociationModel: this,
		Kind:             "belongs_to",
		FieldName:        "Parent",
	}

	var fkFields []*StructField

	for _, f := range primaryFields {
		if fkField, ok := ms.FieldsByName["Parent"+f.Name]; !ok {
			return fmt.Errorf("Foreign field %q does exists in %s type", "Parent"+f.Name, ms.Name)
		} else {
			fkFields = append(fkFields, fkField)
			belongsToRel.ForeignFieldNames = append(belongsToRel.ForeignFieldNames, fkField.Name)
			belongsToRel.ForeignDBNames = append(belongsToRel.ForeignDBNames, fkField.DBName)
			belongsToRel.AssociationForeignFieldNames = append(belongsToRel.AssociationForeignFieldNames, f.Name)
			belongsToRel.AssociationForeignDBNames = append(belongsToRel.AssociationForeignDBNames, f.DBName)
		}
	}

	ms.Indexes["!Parent"] = &StructIndex{
		Model:        ms,
		NameTemplate: "ix_TB_parent",
		Fields:       fkFields,
	}

	belongsToField := &StructField{
		Struct: reflect.StructField{
			Name: "Parent",
			Type: reflect.PtrTo(this.Type),
		},
		Name:         "Parent",
		DBName:       "",
		Data:         map[interface{}]interface{}{},
		BaseModel:    ms,
		Model:        this,
		Relationship: belongsToRel,
	}

	ms.FieldsByName[belongsToField.Name] = belongsToField
	ms.RelatedFields = append(ms.RelatedFields, belongsToField)
	ms.Fields = append(ms.Fields, belongsToField)

	onDel := "CASCADE"
	if field.TagSettings.Flag("DEL_SET_NULL") {
		onDel = "SET NULL"
	}
	ms.ForeignKeys = append(ms.ForeignKeys, &ForeignKey{
		Field:    belongsToField,
		OnDelete: onDel,
		OnUpdate: "CASCADE",
		Prepare: func(srcScope, dstScope *Scope, def *ForeignKeyDefinition) {
			def.Name = ForeignKeyNameOf(def.SrcTableName, field.DBName) + "__parent"
		},
	})

	hasManyRel := *belongsToRel
	hasManyRel.Kind = "has_many"

	field.Relationship = &hasManyRel
	this.RelatedFields = append(this.RelatedFields, field)
	return
}
