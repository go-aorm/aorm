package aorm

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
)

// Relationship described the relationship between models
type Relationship struct {
	Model, AssociationModel      *ModelStruct
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
	case "has_many", "has_one":
		return this.AssoctiationFields()
	default:
		return this.ForeignFields()
	}
}

func (this *Relationship) SetRelatedID(record interface{}, id ID) {
	ID := NewId(this.ForeignFields(), id.Values())
	ID.SetTo(record)
}

func (this *Relationship) GetRelatedID(record interface{}) (id ID) {
	var (
		reflectValue = reflect.Indirect(reflect.ValueOf(record))
		fields       = this.RelatedFields()
		values       = make([]IDValuer, len(fields))
	)
	for _, f := range fields {
		value := reflectValue.FieldByName(f.Name)
		vlr, err := f.IDOf(value.Interface())
		if err != nil {
			panic(errors.Wrapf(err, "field %q", f))
		}
		values = append(values, vlr)
	}
	s := StructOf(indirectRealType(reflect.TypeOf(record)))
	return NewId(s.PrimaryFields, values)
}

func (this *Relationship) DefaultRelatedID() (id ID) {
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
}

func (this *Relationship) ForeignFields() []*StructField {
	var fields = make([]*StructField, len(this.ForeignFieldNames))
	var m *ModelStruct
	switch this.Kind {
	case "has_many", "has_one":
		m = this.AssociationModel
	default:
		m = this.Model
	}
	for i, f := range this.ForeignFieldNames {
		fields[i] = m.FieldsByName[f]
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

func (this *Relationship) InstanceToRelatedID(instance *Instance) (id ID) {
	fieldsNames := this.ForeignFieldNames
	var (
		fields = make([]*StructField, len(fieldsNames))
		values = make([]IDValuer, len(fieldsNames))
	)
	for i, name := range fieldsNames {
		f := instance.FieldsMap[name]
		fields[i] = f.StructField
		var err error
		values[i], err = f.StructField.IDOf(f.Field.Interface())
		if err != nil {
			panic(errors.Wrapf(err, "field %q", f))
		}
	}
	return NewId(fields, values)
}
