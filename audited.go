package aorm

import (
	"fmt"
)

type Auditable interface {
	SetCreatedBy(createdBy interface{})
	GetCreatedBy() string
	SetUpdatedBy(updatedBy interface{})
	GetUpdatedBy() *string
}

type Audited struct {
	CreatedByID string
	UpdatedByID *string
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
	Timestamps
}

type SoftDeleteAuditable interface {
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

type SoftDeleteAuditedModel struct {
	AuditedModel
	SoftDeleteAudited
}

func (scope *Scope) GetCurrentUserID() (id string, ok bool) {
	return scope.db.GetCurrentUserID()
}
