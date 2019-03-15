package aorm

import (
	"fmt"
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

	AuditedFields  = append(append([]string{}, AuditedFieldsByID...), AuditedFieldsAt...)
	AuditedColumns = append(append([]string{}, AuditedColumnsByID...), AuditedColumnsAt...)
)

type Auditor interface {
	Timestamper
	SetCreatedBy(createdBy interface{})
	GetCreatedBy() string
	SetUpdatedBy(updatedBy interface{})
	GetUpdatedBy() *string
}

type Audited struct {
	CreatedByID string
	UpdatedByID *string
	Timestamps
}

func (a *Audited) SetCreatedBy(createdBy interface{}) {
	a.CreatedByID = fmt.Sprintf("%v", createdBy)
}

func (a *Audited) GetCreatedBy() string {
	return a.CreatedByID
}

func (a *Audited) SetUpdatedBy(updatedBy interface{}) {
	if updatedBy == nil {
		a.UpdatedByID = nil
	} else {
		v := fmt.Sprintf("%v", updatedBy)
		a.UpdatedByID = &v
	}
}

func (a *Audited) GetUpdatedBy() *string {
	return a.UpdatedByID
}

type AuditedModel struct {
	KeyStringSerial
	Audited
}

func (scope *Scope) GetCurrentUserID() (id string, ok bool) {
	return scope.db.GetCurrentUserID()
}
