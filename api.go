package aorm

type FieldDataType interface {
	GormDataType(dialect Dialect) string
}

type FieldAssigner interface {
	GormAssigner() Assigner
}
