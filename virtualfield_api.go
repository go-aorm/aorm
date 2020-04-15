package aorm

type (
	VirtualFieldsSetter interface {
		SetVirtualField(fieldName string, value interface{})
	}

	VirtualFieldsGetter interface {
		GetVirtualField(fieldName string) (value interface{}, ok bool)
	}

	ModelWithVirtualFields interface {
		VirtualFieldsGetter
		VirtualFieldsSetter
	}
)
