package aorm

type (
	FieldSelector interface {
		Select(field *StructField, scope *Scope, table string) Query
	}

	FieldSelectWraper interface {
		SelectWrap(field *StructField, scope *Scope, expr string) Query
	}
)

type FieldSelectorFunc = func(field *StructField, scope *Scope, expr string) Query
type fieldSelector FieldSelectorFunc

func (f fieldSelector) Select(field *StructField, scope *Scope, table string) Query {
	return f(field, scope, table)
}

func NewFieldSelector(selectorFunc FieldSelectorFunc) FieldSelector {
	return fieldSelector(selectorFunc)
}
