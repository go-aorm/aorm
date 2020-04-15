package aorm

// Struct get value's model struct, relationships based on struct and tag definition
func (scope *Scope) Struct() *ModelStruct {
	if scope.modelStruct == nil {
		ms, err := scope.db.modelStructStorage.GetOrNew(scope.Value)
		if err != nil {
			panic(err)
		}
		scope.modelStruct = ms
	}
	return scope.modelStruct
}

// StructOf get value's model struct, relationships based on struct and tag definition for value
func (scope *Scope) StructOf(value interface{}) *ModelStruct {
	return scope.New(value).Struct()
}

// Instance get value's instance
func (scope *Scope) Instance() *Instance {
	if scope.instance == nil {
		scope.instance = scope.Struct().InstanceOf(scope.Value)
	}
	return scope.instance
}

// ID get value's id
func (scope *Scope) ID() ID {
	if scope.instance == nil {
		return scope.Struct().GetID(scope.Value)
	}
	return scope.instance.ID()
}

// GetStructFields get model's field structs
func (scope *Scope) GetStructFields() (fields []*StructField) {
	return scope.Struct().Fields
}

// GetNonIgnoredStructFields get non ignored model's field structs
func (scope *Scope) GetNonIgnoredStructFields() []*StructField {
	return scope.Struct().NonIgnoredStructFields()
}

// GetNonIgnoredStructFields get non ignored model's field structs
func (scope *Scope) GetNonRelatedStructFields() []*StructField {
	return scope.Struct().NonRelatedStructFields()
}
