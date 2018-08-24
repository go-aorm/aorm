package aorm

import (
	"fmt"
)

type KeyInterface interface {
	fmt.Stringer
	GetValue() interface{}
}

type key struct {
	Value interface{}
}

func (key *key) String() string {
	switch kt := key.Value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", key)
	case string:
		return kt
	default:
		return fmt.Sprint(kt)
	}
}

func (key *key) GetValue() interface{} {
	return key.Value
}

func Key(value interface{}) KeyInterface {
	return &key{value}
}
