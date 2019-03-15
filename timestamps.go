package aorm

import "time"

const (
	TimestampFieldCreatedAt = "CreatedAt"
	TimestampFieldUpdatedAt = "UpdatedAt"

	TimestampColumnCreatedAt = "created_at"
	TimestampColumnUpdatedAt = "updated_at"
)

var (
	TimestampFields = []string{
		TimestampFieldCreatedAt,
		TimestampFieldUpdatedAt,
	}

	TimestampColumns = []string{
		TimestampColumnCreatedAt,
		TimestampColumnUpdatedAt,
	}
)

type Timestamper interface {
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

type Timestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (t *Timestamps) GetCreatedAt() time.Time {
	return t.CreatedAt
}

func (t *Timestamps) GetUpdatedAt() time.Time {
	return t.UpdatedAt
}
