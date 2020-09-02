package aorm

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
)

// Relationship described the relationship between models
type Relationship struct {
	Model, AssociationModel      *ModelStruct
	Field                        *StructField
	Kind                         string
	FieldName                    string
	PolymorphicType              string
	PolymorphicDBName            string
	PolymorphicValue             func(ctx context.Context, singular bool) string
	ForeignFieldNames            []string
	ForeignDBNames               []string
	AssociationForeignFieldNames []string
	AssociationForeignDBNames    []string
	JoinTableHandler             JoinTableHandlerInterface
}

func (this *Relationship) RelatedFields() []*StructField {
	switch this.Kind {
	case "belongs_to", "has_one":
		return this.AssoctiationFields()
	default:
		return this.ForeignFields()
	}
}

func (this *Relationship) SetRelatedID(record interface{}, id ID) {
	switch this.Kind {
	case "belongs_to", "has_many":
		ID := NewId(this.ForeignFields(), id.Values())
		ID.SetTo(record)
	default:
		panic("not implemented")
	}
}

func (this *Relationship) GetRelatedID(record interface{}) (id ID) {
	switch this.Kind {
	case "belongs_to", "has_many":
		var (
			reflectValue = reflect.Indirect(reflect.ValueOf(record))
			values       = make([]IDValuer, len(this.ForeignFieldNames))
		)

		for i, fieldName := range this.ForeignFieldNames {
			structField := this.Model.FieldsByName[fieldName]
			fieldValue := reflectValue.FieldByIndex(structField.StructIndex)
			vlr, err := structField.IDOf(fieldValue)
			if err != nil {
				panic(errors.Wrapf(err, "field %q", structField))
			}
			values[i] = vlr
		}
		return NewId(this.AssoctiationFields(), values)
	}
	panic("not implemented")
}

func (this *Relationship) ParseRelatedID(s string) (id ID, err error) {
	switch this.Kind {
	case "belongs_to", "has_many":
		if id, err = this.AssociationModel.ParseIDString(s); err != nil {
			return
		}
		var (
			fields = this.RelatedFields()
			values = id.Values()
		)
		return NewId(fields, values), nil
	default:
		panic("not implemented")
	}
}

func (this *Relationship) ForeignID(relatedID ID) ID {
	var fields = make([]*StructField, len(relatedID.Fields()), len(relatedID.Fields()))
	for i, f := range relatedID.Fields() {
		for j, rf := range this.RelatedFields() {
			if rf.Name == f.Name {
				fields[i] = this.Model.FieldsByName[this.ForeignFieldNames[j]]
				break
			}
		}
	}
	return NewId(fields, relatedID.Values())
}

func (this *Relationship) DefaultRelatedID() (id ID) {
	switch this.Kind {
	case "belongs_to", "has_many":
		var (
			fields = this.RelatedFields()
			values = make([]IDValuer, len(fields))
		)
		for _, f := range fields {
			vlr, err := f.DefaultID()
			if err != nil {
				panic(errors.Wrapf(err, "field %q", f))
			}
			values = append(values, vlr)
		}
		return NewId(fields, values)
	default:
		panic("not implemented")
	}
}

func (this *Relationship) InstanceToRelatedID(instance *Instance) (id ID) {
	switch this.Kind {
	case "belongs_to", "has_many":
		var (
			fields = this.AssoctiationFields()
			values = make([]IDValuer, len(fields))
			err    error
		)
		for i, structField := range fields {
			values[i], err = instance.FieldsMap[structField.Name].ID()
			if err != nil {
				panic(errors.Wrapf(err, "field %q", structField))
			}
		}
		return NewId(fields, values)
	default:
		panic("not implemented")
	}
}

func (this *Relationship) ForeignFields() []*StructField {
	var fields = make([]*StructField, len(this.ForeignFieldNames))
	for i, f := range this.ForeignFieldNames {
		fields[i] = this.Model.FieldsByName[f]
	}
	return fields
}

func (this *Relationship) AssoctiationFields() []*StructField {
	var fields = make([]*StructField, len(this.AssociationForeignFieldNames))
	for i, f := range this.AssociationForeignFieldNames {
		fields[i] = this.AssociationModel.FieldsByName[f]
	}
	return fields
}

func (this Relationship) Copy() *Relationship {
	if this.JoinTableHandler != nil {
		this.JoinTableHandler = this.JoinTableHandler.Copy()
	}
	return &this
}
