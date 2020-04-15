package aorm

import (
	"github.com/moisespsena-go/bid"
)

// Model base model definition, including fields `BID`, `CreatedAt`, `UpdatedAt`, `DeletedAt`, which could be embedded in your models
//    type User struct {
//      aorm.Model
//    }
type Model struct {
	ID bid.BID `aorm:"primary_key;serial"`
}

type ModelTS struct {
	Model
	Timestamps
}
