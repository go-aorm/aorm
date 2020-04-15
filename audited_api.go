package aorm

type Auditor interface {
	Timestamper
	SetCreatedBy(createdBy IDValuer)
	GetCreatedBy() IDValuer
	SetUpdatedBy(updatedBy IDValuer)
	GetUpdatedBy() IDValuer
}
