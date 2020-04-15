package aorm

import (
	"github.com/moisespsena-go/bid"
)

const (
	AuditedFieldCreatedByID = "CreatedByID"
	AuditedFieldUpdatedByID = "UpdatedByID"

	AuditedColumnCreatedByID = "created_by_id"
	AuditedColumnUpdatedByID = "updated_by_id"

	AuditedFieldCreatedAt = "CreatedAt"
	AuditedFieldUpdatedAt = "UpdatedAt"

	AuditedColumnCreatedAt = "created_at"
	AuditedColumnUpdatedAt = "updated_at"
)

var (
	AuditedFieldsByID = []string{
		AuditedFieldCreatedByID,
		AuditedFieldUpdatedByID,
	}

	AuditedFieldsAt = append([]string{
		AuditedFieldCreatedAt,
		AuditedFieldUpdatedAt,
	}, TimestampFields...)

	AuditedColumnsByID = []string{
		AuditedColumnCreatedByID,
		AuditedColumnUpdatedByID,
	}

	AuditedColumnsAt = append([]string{
		AuditedColumnCreatedAt,
		AuditedColumnUpdatedAt,
	}, TimestampColumns...)

	AuditedFields  = append(AuditedFieldsByID, AuditedFieldsAt...)
	AuditedColumns = append(AuditedColumnsByID, AuditedColumnsAt...)
)

type Audited struct {
	CreatedByID bid.BID
	UpdatedByID bid.BID
	Timestamps
}

func (a *Audited) SetCreatedBy(value IDValuer) {
	if value == nil {
		a.CreatedByID = bid.Zero()
	} else {
		a.CreatedByID = value.Raw().(bid.BID)
	}
}

func (a *Audited) GetCreatedBy() IDValuer {
	return BytesId(a.CreatedByID[:])
}

func (a *Audited) SetUpdatedBy(value IDValuer) {
	if value == nil {
		a.UpdatedByID = bid.Zero()
	} else {
		a.UpdatedByID = value.Raw().(bid.BID)
	}
}

func (a *Audited) GetUpdatedBy() IDValuer {
	return BytesId(a.UpdatedByID[:])
}

type AuditedModel struct {
	Model
	Audited
}

func (scope *Scope) GetCurrentUserID() (id interface{}) {
	return scope.db.GetCurrentUserID()
}
