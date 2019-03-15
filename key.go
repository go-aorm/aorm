package aorm

import (
	"fmt"
	"strings"
)

type KeyInterface interface {
	fmt.Stringer
	Values() []interface{}
	Strings() []string
	Append(value ...interface{}) KeyInterface
}

type key struct {
	values []interface{}
}

func (key *key) Append(value ...interface{}) KeyInterface {
	key.values = append(key.values, value...)
	return key
}

func (key *key) Strings() []string {
	var s = make([]string, len(key.values))
	for i, v := range key.values {
		if v == nil {
			continue
		}
		switch kt := v.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			if kt != 0 {
				s[i] = fmt.Sprintf("%d", key)
			} else {
				s[i] = ""
			}
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
