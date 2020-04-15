package aorm

import (
	"encoding/base64"
)

type BytesId []byte

func (this BytesId) Bytes() []byte {
	return this
}

func (BytesId) ParseBytes(b []byte) (IDValuer, error) {
	vlr := make([]byte, len(b))
	copy(vlr, b)
	return BytesId(vlr), nil
}

func (BytesId) ParseString(s string) (_ IDValuer, err error) {
	var b BytesId
	b, err = base64.RawURLEncoding.DecodeString(s)
	return b, nil
}

func (this BytesId) String() string {
	return base64.RawURLEncoding.EncodeToString(this)
}

func (this BytesId) IsZero() bool {
	return len(this) == 0
}

func (this BytesId) Raw() interface{} {
	return []byte(this)
}
