package aorm

type (
	TableNamer interface {
		TableName() string
	}

	TableNamePlurabler interface {
		TableName(singular bool) string
	}

	TableNameResolver interface {
		TableName(singular bool, value interface{}) (tableName string, ok bool)
	}
)
