package aorm

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Id struct {
	TableName string
	fields    []*StructField
	values    []IDValuer
	exclude   bool
}

func NewValuedId(values ...IDValuer) *Id {
	return &Id{fields: make([]*StructField, len(values), len(values)), values: values}
}

func NewId(fields []*StructField, values []IDValuer) *Id {
	return &Id{fields: fields, values: values}
}

func (this Id) SetValue(value ...interface{}) (_ ID, err error) {
	if l := len(this.values); l == len(value) {
		this.values = make([]IDValuer, l, l)
		for i, v := range value {
			switch t := v.(type) {
			case IDValuer:
				this.values[i] = t
			default:
				if this.values[i], err = this.fields[i].IDOf(v); err != nil {
					return nil, err
				}
			}
		}
		return this, nil
	} else {
		return nil, fmt.Errorf("ID require " + strconv.Itoa(len(this.values)) + " values to set")
	}
}

func (this Id) Bytes() (b []byte) {
	for _, v := range this.values {
		vb := v.Bytes()
		b = append(b, uint8(len(vb)))
		b = append(b, vb...)
	}

	return b
}

func (this Id) String() string {
	if this.IsZero() {
		return ""
	}
	if len(this.values) == 1 {
		return this.values[0].String()
	}
	return base64.RawURLEncoding.EncodeToString(this.Bytes())
}

func (this Id) IsZero() bool {
	if len(this.values) == 0 {
		return true
	}
	for _, v := range this.values {
		if v == nil || v.IsZero() {
			return true
		}
	}
	return false
}

func (this Id) Fields() []*StructField {
	return this.fields
}

func (this Id) Values() []IDValuer {
	return this.values
}

func (this Id) SetTo(recorde interface{}) interface{} {
	var (
		rv reflect.Value
		ok bool
	)
	if rv, ok = recorde.(reflect.Value); !ok {
		rv = reflect.ValueOf(recorde)
	}
	rv = indirect(rv)

	for i, f := range this.Fields() {
		frv := rv.FieldByIndex(f.StructIndex)
		if this.values[i] == nil {
			SetZero(frv)
		} else {
			SetNonZero(frv, reflect.ValueOf(this.values[i].Raw()).Convert(frv.Type()).Interface())
		}
	}
	return recorde
}

func (this Id) WhereClause(scope *Scope) (result Query) {
	tbName := this.TableName
	if tbName == "" {
		tbName = scope.QuotedTableName()
	}
	if tbName != "" {
		tbName += "."
	}

	if this.exclude && len(this.fields) == 1 {
		result.Query = tbName + this.fields[0].DBName + " != ?"
		result.AddArgs(this.values[0].Raw())
		return
	}

	var q []string
	for i, f := range this.fields {
		q = append(q, tbName+f.DBName+" = ?")
		result.AddArgs(this.values[i].Raw())
	}
	result.Query = strings.Join(q, " AND ")
	return
}

func (this Id) Exclude() ID {
	this.exclude = true
	return this
}

type idSliceScoper struct {
	exclude bool
	values  []ID
}

func (this idSliceScoper) Values() []ID {
	return this.values
}

func (this idSliceScoper) Exclude() idSliceScoper {
	this.exclude = true
	return this
}

func (this idSliceScoper) WhereClause(scope *Scope) (result Query) {
	tbName := scope.TableName() + "."
	var q []string
	for _, id := range this.values {
		switch t := id.(type) {
		case DBNamer:
			q = append(q, tbName+t.DBName()+" = ?")
			result.AddArgs(id)
		case WhereClauser:
			c := t.WhereClause(scope)
			q = append(q, c.Query)
			result.AddArgs(c.Args...)
		default:
			q = append(q, tbName+"id = ?")
			result.AddArgs(id)
		}
	}
	var op string
	if this.exclude {
		op = "NOT "
	}
	result.Query = op + "(" + strings.Join(q, " OR ") + ")"
	return
}

type idSliceNamedTable struct {
	TableName string
	exclude   bool
	values    []ID
}

func (this idSliceNamedTable) Values() []ID {
	return this.values
}

func (this idSliceNamedTable) Exclude() idSliceNamedTable {
	this.exclude = true
	return this
}

func (this idSliceNamedTable) WhereClause(scope *Scope) (result Query) {
	var q []string
	for _, id := range this.values {
		c := id.WhereClause(scope)
		q = append(q, c.Query)
		result.AddArgs(c.Args...)
	}
	var op string
	if this.exclude {
		op = "NOT "
	}
	result.Query = op + "(" + strings.Join(q, " OR ") + ")"
	return
}
