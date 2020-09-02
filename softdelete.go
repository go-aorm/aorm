package aorm

import (
	"time"

	"github.com/moisespsena-go/bid"
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

func IsAuditedSdField(fieldName string) bool {
	for _, name := range AuditedSDFields {
		if name == fieldName {
			return true
		}
	}
	return false
}

type SoftDelete struct {
	DeletedAt *time.Time `sql:"index"`
}

func (d *SoftDelete) GetDeletedAt() *time.Time {
	return d.DeletedAt
}

type SoftDeleteAudited struct {
	SoftDelete
	DeletedByID bid.BID
}

func (a *SoftDeleteAudited) SetDeletedBy(deletedBy interface{}) {
	a.DeletedByID = bid.From(deletedBy)
}

func (a *SoftDeleteAudited) GetDeletedBy() interface{} {
	return a.DeletedByID
}

type AuditedSD struct {
	Audited
	SoftDeleteAudited
}

type AuditedSDModel struct {
	Model
	AuditedSD
}
