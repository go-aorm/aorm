package aorm

// ModelStructStorage get the model struct storage fom DB
func (s *DB) ModelStructStorage() *ModelStructStorage {
	return s.modelStructStorage
}

// StructOf get value's model struct, relationships based on struct and tag definition fom DB
func (s *DB) StructOf(value ...interface{}) *ModelStruct {
	var v = s.Val
	for _, v = range value {
	}

	ms, err := s.modelStructStorage.GetOrNew(v)
	if err != nil {
		panic(err)
	}
	return ms
}
