package aorm

import (
	"encoding/binary"
	"fmt"
	"strconv"
)

type IntId int64

func (this IntId) Int() int64 {
	return int64(this)
}

func (this IntId) Bytes() []byte {
	var b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(this))
	return b
}

func (this IntId) ParseBytes(b []byte) (IDValuer, error) {
	switch len(b) {
	case 0:
		this = 0
	case 1:
		this = IntId(b[0])
	case 2:
		this = IntId(binary.BigEndian.Uint16(b))
	case 4:
		this = IntId(binary.BigEndian.Uint32(b))
	case 8:
		this = IntId(binary.BigEndian.Uint64(b))
	}
	return this, nil
}

func (this IntId) ParseString(s string) (_ IDValuer, err error) {
	var i int64
	if i, err = strconv.ParseInt(s, 10, 64); err != nil {
		return
	}
	return IntId(i), nil
}

func (this IntId) String() string {
	return fmt.Sprint(this)
}

func (this IntId) IsZero() bool {
	return this == 0
}

func (this IntId) Raw() interface{} {
	return int64(this)
}
