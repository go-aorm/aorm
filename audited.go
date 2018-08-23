package aorm

import (
	"fmt"
	"strconv"
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

func getCurrentUser(scope *Scope) (string, bool) {
	var user interface{}
	var hasUser bool

	user, hasUser = scope.DB().Get("gorm:current_user")

	if !hasUser {
		return "", false
	}

	var currentUser string
	switch ut := user.(type) {
	case string:
		return ut, currentUser != ""
	case uint:
		return strconv.Itoa(int(ut)), ut != 0
	default:
		if primaryField := scope.New(user).PrimaryField(); primaryField != nil {
			currentUser = fmt.Sprintf("%v", primaryField.Field.Interface())
		} else {
			currentUser = fmt.Sprintf("%v", user)
		}
		return currentUser, true
	}
}
