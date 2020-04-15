package aorm

import (
	"reflect"

	"github.com/moisespsena-go/bid"
)

var bidType = reflect.TypeOf(bid.BID{})

type DefinedID struct {
	ID bid.BID `aorm:"primary_key"`
}
