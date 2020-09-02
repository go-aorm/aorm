package types

import (
	"bytes"
	"database/sql/driver"
	"encoding/csv"
	"errors"
	"reflect"

	"github.com/moisespsena-go/aorm"
)

type (
	Strings []string

	StringsAssigner struct {
	}
)

func (this Strings) Value() (driver.Value, error) {
	var (
		w  bytes.Buffer
		cw = csv.NewWriter(&w)
	)
	if err := cw.Write(this); err != nil {
		return nil, err
	}
	cw.Flush()
	return w.String(), nil
}

func (this *Strings) Scan(src interface{}) (err error) {
	*this = nil
	if src == nil {
		return nil
	}
	var s string
	switch t := src.(type) {
	case string:
		s = t
		goto set
	case []byte:
		s = string(t)
		goto set
	default:
		return errors.New("bad source type")
	}
set:
	if s == "" {
		return
	}
	var v []string
	if v, err = csv.NewReader(bytes.NewBufferString(s)).Read(); err == nil {
		*this = v
	}
	return
}

func init() {
	aorm.Register(StringsAssigner{})
}

func (this Strings) PrimaryGoValue() interface{} {
	return []string{}
}

func (this Strings) IsZero() bool {
	return len(this) == 0
}

func (StringsAssigner) Valuer(_ aorm.Dialector, value interface{}) driver.Valuer {
	return value.(Strings)
}

func (StringsAssigner) Scaner(_ aorm.Dialector, dest reflect.Value) aorm.Scanner {
	return dest.Addr().Interface().(*Strings)
}

func (StringsAssigner) SQLType(aorm.Dialector) string {
	return "TEXT"
}

func (StringsAssigner) SQLSize(_ aorm.Dialector) int {
	return 0
}

func (StringsAssigner) Type() reflect.Type {
	return reflect.TypeOf(Strings{})
}
