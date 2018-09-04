package aorm

type VirtualFieldsSetter interface {
	SetVirtualField(fieldName string, value interface{})
}

type VirtualFieldsGetter interface {
	GetVirtualField(fieldName string) (value interface{}, ok bool)
}

type ModelWithVirtualFields interface {
	VirtualFieldsGetter
	VirtualFieldsSetter
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
