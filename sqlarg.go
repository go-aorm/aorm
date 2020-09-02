package aorm

import (
	"bytes"
	"strconv"
)

type UintTuple []uint64

func (i UintTuple) SqlArg() (value interface{}) {
	var b bytes.Buffer
	b.WriteByte('(')
	for _, v := range i {
		b.WriteString(strconv.FormatUint(v, 10) + ",")
	}
	data := b.Bytes()
	// skip last ","
	data = data[0 : len(data)-1]
	return RawSqlArg(string(data) + ")")
}
