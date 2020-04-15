package aorm

// StructOf get value's model struct, relationships based on struct and tag definition for value
func StructOf(value interface{}) *ModelStruct {
	ms, _ := modelStructStorage.GetOrNew(value)
	return ms
}

// PrimaryFieldsOf get the primary fields instance of value
func PrimaryFieldsOf(value interface{}) []*Field {
	return StructOf(value).PrimaryFieldsInstance(value)
}

// IdOf get ID of model value
func IdOf(value interface{}) ID {
	return StructOf(value).GetID(value)
}

// IdStringTo parses id from strign and set it to model value
func IdStringTo(id string, value interface{}) error {
	return StructOf(value).SetIdFromString(value, id)
}

// ZeroIdOf returns if ID of model value is zero
func ZeroIdOf(value interface{}) bool {
	id := IdOf(value)
	return id == nil || id.IsZero()
}

// InstanceOf get instance of model value
func InstanceOf(value interface{}, fieldsNames ...string) *Instance {
	return StructOf(value).InstanceOf(value, fieldsNames...)
}
