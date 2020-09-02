package aorm

type FieldFlag uint16

func (this FieldFlag) Set(flag FieldFlag) FieldFlag    { return this | flag }
func (this FieldFlag) Clear(flag FieldFlag) FieldFlag  { return this &^ flag }
func (this FieldFlag) Toggle(flag FieldFlag) FieldFlag { return this ^ flag }
func (this FieldFlag) Has(flag FieldFlag) bool         { return this&flag != 0 }

const (
	FieldEmptyFlag FieldFlag = 1 << iota
	FieldCreationStoreEmpty
	FieldUpdationStoreEmpty
)
