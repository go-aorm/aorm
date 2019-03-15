package aorm

import (
	"fmt"
	"time"
)

const (
	SoftDeleteFieldDeletedByID = "DeletedByID"
	SoftDeleteFieldDeletedAt   = "DeletedAt"

	SoftDeletedColumnDeletedByID = "deleted_by_id"
	SoftDeleteColumnDeletedAt    = "deleted_at"
)

var (
	SoftDeleteFields = []string{
		SoftDeleteFieldDeletedByID,
		SoftDeleteFieldDeletedAt,
	}

	AuditedSDFields = append(append([]string{}, AuditedFields...), SoftDeleteFields...)
)

type SoftDeleter interface {
	GetDeletedAt() *time.Time
}

type SoftDelete struct {
	DeletedAt *time.Time `sql:"index"`
}

func (d *SoftDelete) GetDeletedAt() *time.Time {
	return d.DeletedAt
}

type SoftDeleteAuditor interface {
	SoftDeleter
	SetDeletedBy(deletedBy interface{})
	GetDeletedBy() *string
}

type SoftDeleteAudited struct {
	SoftDelete
	DeletedByID *string
}

func (a *SoftDeleteAudited) SetDeletedBy(deletedBy interface{}) {
	if deletedBy == nil {
		a.DeletedByID = nil
	} else {
		v := fmt.Sprintf("%v", deletedBy)
		a.DeletedByID = &v
	}
}

func (a *SoftDeleteAudited) GetDeletedBy() *string {
	return a.DeletedByID
}

type AuditedSD struct {
	Audited
	SoftDeleteAudited
}

type AuditedSDModel struct {
	KeyStringSerial
	AuditedSD
}
