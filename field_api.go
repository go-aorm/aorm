package aorm

type (
	DBNamer interface {
		DBName() string
	}

	CanChilder interface {
		AormCanChild() bool
	}
)
