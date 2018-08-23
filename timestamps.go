package aorm

import "time"

type Timestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SoftDelete struct {
	DeletedAt *time.Time `sql:"index"`
}
