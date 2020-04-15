package aorm

type StructFieldSetuper interface {
	AormStructFieldSetup(model *ModelStruct, field *StructField)
}
