package aorm

type (
	TableNamePrefixer interface {
		TableNamePrefix() string
	}
)
