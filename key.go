package aorm

import (
	"fmt"
	"strings"
)

type KeyInterface interface {
	fmt.Stringer
	Values() []interface{}
	Strings() []string
}

type key struct {
	values []interface{}
}

func (key *key) Strings() []string {
	var s = make([]string, len(key.values))
	for i, v := range key.values {
		switch kt := v.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			s[i] = fmt.Sprintf("%d", key)
		case string:
			s[i] = kt
		default:
			s[i] = fmt.Sprint(kt)
		}
	}
	return s
}

func (key *key) String() string {
	return strings.Join(key.Strings(), ",")
}

func (key *key) Values() []interface{} {
	return key.values
}

func Key(value ...interface{}) KeyInterface {
	return &key{value}
}
