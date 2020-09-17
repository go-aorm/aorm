package aorm

import (
	"github.com/jinzhu/inflection"
)

func (this *ModelStruct) addChildField(field *StructField, childModel *ModelStruct) (err error) {
	childModel = childModel.Clone()
	field.Model = childModel

	// reset table names
	childModel.PluralTableName = ""
	childModel.SingularTableName = ""

	var name, _ = ChildName(field)
	field.DBName = ToDBName(name)

	childModel.Name = name
	childModel.Parent = this
	childModel.ParentField = field

	this.ChildrenByName[field.Name] = childModel
	this.Children = append(this.Children, childModel)

	if len(this.ChildrenByName) != len(this.Children) {
		panic("bad langth of children")
	}

	field.IsNormal = false
	field.IsChild = true

	relationship := &Relationship{
		Field:            field,
		Model:            this,
		AssociationModel: childModel,
		Kind:             "has_one",
	}

	for _, f := range childModel.PrimaryFields {
		relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, f.Name)
		relationship.ForeignDBNames = append(relationship.ForeignDBNames, f.DBName)
		relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, f.Name)
		relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, f.DBName)
	}

	field.Relationship = relationship
	this.InlinePreloadFields = append(this.InlinePreloadFields, field.Name)
	this.RelatedFields = append(this.RelatedFields, field)
	childModel.ParentForeignKey = &ForeignKey{
		Field:    field,
		OnDelete: "CASCADE",
		OnUpdate: "CASCADE",
		Prepare: func(srcScope, dstScope *Scope, def *ForeignKeyDefinition) {
			def.Name = ForeignKeyNameOf(def.SrcTableName, "parent")
		},
	}
	childModel.ForeignKeys = append(childModel.ForeignKeys, childModel.ParentForeignKey)

	for i, child := range childModel.Children {
		child = child.Clone()
		child.Parent = childModel
		childModel.Children[i] = child
	}

	return
}

func ChildName(field *StructField) (singular, plural string) {
	if v := field.TagSettings["CHILD"]; v != "" && v != "CHILD" {
		if field.TagSettings.Scanner().IsTags(v) {
			tags := field.TagSettings.TagsOf(v)
			singular = tags["S"]
			if plural = tags["P"]; plural == "." {
				plural = inflection.Plural(singular)
			}
		} else {
			singular = v
		}
	}

	if singular == "" || plural == "" && field.Model.Tags != nil {
		if v := field.Model.Tags["CHILD"]; v != "" && v != "CHILD" {
			if field.Model.Tags.Scanner().IsTags(v) {
				tags := field.TagSettings.TagsOf(v)
				if singular == "" {
					singular = tags["S"]
				}
				if plural == "" {
					plural = tags["P"]
				}
			} else if singular == "" {
				singular = v
			}
		}
	}

	if singular == "" {
		singular = field.Name
	}
	if plural == "" {
		plural = inflection.Plural(singular)
	}
	return
}
