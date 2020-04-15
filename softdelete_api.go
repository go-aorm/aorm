package aorm

import "time"

type (
	SoftDeleter interface {
		GetDeletedAt() *time.Time
	}

	SoftDeleteAuditor interface {
		SoftDeleter
		SetDeletedBy(deletedBy interface{})
		GetDeletedBy() *string
	}
)
