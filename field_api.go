package aorm

type (
	DBNamer interface {
		DBName() string
	}
)
