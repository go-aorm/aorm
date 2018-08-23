package aorm

import "strconv"

type ModelInterface interface {
	GetID() string
	SetID(v string)
}

// Model base model definition, including fields `ID`, `CreatedAt`, `UpdatedAt`, `DeletedAt`, which could be embedded in your models
//    type User struct {
//      aorm.Model
//    }
type Model struct {
	ID uint `gorm:"primary_key"`
}

func (m *Model) GetID() string {
	return strconv.Itoa(int(m.ID))
}

func (m *Model) SetID(v string) {
	id, _ := strconv.Atoi(v)
	m.ID = uint(id)
}

type ModelTS struct {
	Model
	Timestamps
}
