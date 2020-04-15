package aorm

type StrId string

func (this StrId) Bytes() []byte {
	return []byte(this)
}

func (StrId) ParseBytes(b []byte) (IDValuer, error) {
	return StrId(string(b)), nil
}

func (StrId) ParseString(s string) (IDValuer, error) {
	return StrId(s), nil
}

func (this StrId) String() string {
	return string(this)
}

func (this StrId) IsZero() bool {
	return string(this) == ""
}

func (this StrId) Raw() interface{} {
	return string(this)
}
