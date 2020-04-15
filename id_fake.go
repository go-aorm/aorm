package aorm

type fakeId string

func FakeID(v string) ID {
	return fakeId(v)
}

func (this fakeId) IsZero() bool {
	return this == ""
}

func (this fakeId) WhereClause(scope *Scope) (result Query) {
	panic("fake id: not implemented")
}

func (this fakeId) Exclude() ID {
	panic("fake id: not implemented")
}

func (this fakeId) String() string {
	return string(this)
}

func (this fakeId) Bytes() []byte {
	panic("fake id: not implemented")
}

func (this fakeId) Fields() []*StructField {
	panic("fake id: not implemented")
}

func (this fakeId) Field() *StructField {
	panic("fake id: not implemented")
}

func (this fakeId) Values() []IDValuer {
	return []IDValuer{StrId(this)}
}

func (this fakeId) Value() IDValuer {
	return StrId(this)
}

func (this fakeId) Raw() interface{} {
	return string(this)
}

func (this fakeId) SetValue(value ...interface{}) (ID, error) {
	switch t := value[0].(type) {
	case StrId:
		this = fakeId(t)
	default:
		this = fakeId(value[0].(string))
	}
	return this, nil
}

func (this fakeId) SetTo(recorde interface{}) interface{} {
	panic("fake id: not implemented")
}
