package aorm

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
)

type UintId uint64

func (this UintId) Uint() uint64 {
	return uint64(this)
}

func (this UintId) Bytes() []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(this))
	return b[:]
}

func (UintId) ParseBytes(b []byte) (IDValuer, error) {
	switch len(b) {
	case 0:
		return UintId(0), nil
	case 1:
		return UintId(b[0]), nil
	case 2:
		return UintId(binary.BigEndian.Uint16(b)), nil
	case 4:
		return UintId(binary.BigEndian.Uint32(b)), nil
	case 8:
		return UintId(binary.BigEndian.Uint64(b)), nil
	default:
		return nil, errors.New("bad uint bytes size")
	}
}

func (UintId) ParseString(s string) (_ IDValuer, err error) {
	var i uint64
	if i, err = strconv.ParseUint(s, 10, 64); err != nil {
		return
	}
	return UintId(i), nil
}

func (this UintId) String() string {
	return fmt.Sprint(uint64(this))
}

func (this UintId) IsZero() bool {
	return this == 0
}

func (this UintId) Raw() interface{} {
	return uint64(this)
}
