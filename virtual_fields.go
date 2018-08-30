package aorm

type ModelWithVirtualFields interface {
	SetVirtualField(fieldName string, value interface{})
	GetVirtualField(fieldName string) (value interface{}, ok bool)
}

type VirtualField struct {
	ModelStruct *ModelStruct
	FieldName   string
	StructIndex int
	Value       interface{}
	Options     map[interface{}]interface{}
	Setter      func(recorde, value interface{})
	Getter      func(recorde interface{}) (value interface{}, ok bool)
}
