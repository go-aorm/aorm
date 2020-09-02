package aorm

type VirtualFields struct {
	values map[string]interface{}
}

func (m *VirtualFields) GetVirtualField(fieldName string) (value interface{}, ok bool) {
	if m.values == nil {
		return
	}
	value, ok = m.values[fieldName]
	return
}

func (m *VirtualFields) SetVirtualField(fieldName string, value interface{}) {
	if m.values == nil {
		m.values = map[string]interface{}{}
	}
	m.values[fieldName] = value
}

type VirtualField struct {
	Model                *ModelStruct
	FieldName            string
	StructIndex          int
	Value                interface{}
	Options              map[interface{}]interface{}
	Setter               func(vf *VirtualField, recorde, value interface{})
	Getter               func(vf *VirtualField, recorde interface{}) (value interface{}, ok bool)
	LocalFieldName       string
	InlinePreloadOptions InlinePreloadOptions
}

type VirtualFieldValue struct {
	Value interface{}
}

func (vf *VirtualField) Set(recorde, value interface{}) {
	if vf.Setter != nil {
		vf.Setter(vf, recorde, value)
	} else {
		recorde.(VirtualFieldsSetter).SetVirtualField(vf.FieldName, value)
	}
}

func (vf *VirtualField) Get(recorde interface{}) (value interface{}, ok bool) {
	if vf.Getter != nil {
		return vf.Getter(vf, recorde)
	}
	return recorde.(VirtualFieldsGetter).GetVirtualField(vf.FieldName)
}
