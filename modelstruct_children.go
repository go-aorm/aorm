package aorm

import (
	"reflect"
	"strings"
)

func (this *ModelStruct) addChildField(pth []string, field *StructField) (err error) {
	var (
		ms  *ModelStruct
		typ = indirectType(field.Struct.Type)
	)

	if ms, err = this.storage.create(typ); err != nil {
		return
	}

	this.storage.ModelStructsMap.delete(typ)
	pth = append(pth, field.Name)
	ms.Name = strings.Join(pth, "")
	ms.Parent = this
	ms.ParentField = field

	if this.Root != nil {
		ms.Root = this.Root
	} else {
		ms.Root = this
	}

	this.ChildrenByName[field.Name] = ms
	this.Children = append(this.Children, ms)

	if len(this.ChildrenByName) != len(this.Children) {
		panic("bad langth of children")
	}

	field.Model = ms
	field.IsNormal = false
	field.IsChild = true

	primaryFields := this.PrimaryFields
	if this.Parent != nil {
		primaryFields = this.Root.PrimaryFields
	}

	for _, f := range primaryFields {
		cf := &StructField{
			Struct: reflect.StructField{
				Name: f.Name,
				Type: f.Struct.Type,
			},
			Name:   f.Name,
			DBName: f.DBName,
			Tag:    f.Tag,
			TagSettings: map[string]string{
				"AUTO_INCREMENT_DISABLED": "AUTO_INCREMENT_DISABLED",
			},
			StructIndex:  nil,
			Data:         map[interface{}]interface{}{},
			BaseModel:    ms,
			IsPrimaryKey: true,
			IsNormal:     true,
			Assigner:     f.Assigner,
		}
		ms.FieldsByName[cf.Name] = cf
		ms.PrimaryFields = append(ms.PrimaryFields, cf)
		ms.Fields = append(ms.Fields, cf)
	}

	relationship := &Relationship{
		Model:            this,
		AssociationModel: ms,
		Kind:             "has_one",
	}

	for i, f := range primaryFields {
		cf := ms.PrimaryFields[i]
		relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, f.Name)
		relationship.ForeignDBNames = append(relationship.ForeignDBNames, f.DBName)
		relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, cf.Name)
		relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, cf.DBName)
	}

	field.Relationship = relationship
	this.InlinePreloadFields = append(this.InlinePreloadFields, field.Name)
	this.RelatedFields = append(this.RelatedFields, field)
	ms.ForeignKeys = append(ms.ForeignKeys, &ForeignKey{
		Field:    field,
		OnDelete: "CASCADE",
		OnUpdate: "CASCADE",
		Prepare: func(srcScope, dstScope *Scope, def *ForeignKeyDefinition) {
			def.Name = ForeignKeyNameOf(def.SrcTableName, field.DBName) + "__parent"
		},
	})
	err = ms.setup(nil, false, nil)
	return
}
