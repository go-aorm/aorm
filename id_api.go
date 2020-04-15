package aorm

import "fmt"

type (
	IDValuer interface {
		Bytes() []byte
		fmt.Stringer
		Zeroer
		Raw() interface{}
		ParseString(s string) (value IDValuer, err error)
		ParseBytes(s []byte) (value IDValuer, err error)
	}

	ID interface {
		Zeroer
		WhereClauser
		fmt.Stringer
		Bytes() []byte
		Fields() []*StructField
		Field() *StructField
		Values() []IDValuer
		Value() IDValuer
		Raw() interface{}
		SetValue(value ...interface{}) (ID, error)
		SetTo(recorde interface{}) interface{}
		Exclude() ID
	}

	IDSlicer interface {
		Values() []ID
		Exclude() IDSlicer
	}

	IDValueRawConverter interface {
		FromRaw(raw interface{}) IDValuer
		ToRaw(value IDValuer) interface{}
	}
)
