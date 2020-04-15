package aorm

import (
	"encoding/base64"
	"reflect"

	"github.com/pkg/errors"
)

var idValueRawConverterMap syncedMap

func IDValueRawConverterRegister(typ reflect.Type, converter IDValueRawConverter) {
	idValueRawConverterMap.Set(typ, converter)
}

func IDValueRawConverterGet(typ reflect.Type) (converter IDValueRawConverter) {
	if v, ok := idValueRawConverterMap.Get(typ); ok {
		return v.(IDValueRawConverter)
	}
	return
}

func (this *ModelStruct) ParseIDBytes(b []byte) (_ ID, err error) {
	if len(b) == 0 {
		return nil, errors.New("parse id from bytes: empty byte slice")
	}
	var (
		fields = this.PrimaryFields
		values = make([]IDValuer, len(fields), len(fields))
	)
	if len(fields) == 1 {
		if values[0], err = fields[0].DefaultID(); err != nil {
			return
		}
		if values[0], err = values[0].ParseBytes(b); err != nil {
			return nil, errors.Wrap(err, "aorm.IdValuer.ParseBytes")
		}
	} else {
		for i, fi := 0, 0; i < len(b); fi++ {
			if values[fi], err = fields[fi].DefaultID(); err != nil {
				return
			}
			tmp := b[i+1 : i+int(b[i])+1]
			if values[fi], err = values[fi].ParseBytes(tmp); err != nil {
				return nil, errors.Wrap(err, "aorm.IdValuer.ParseBytes")
			}
			i += int(b[i] + 1)
		}
	}
	return NewId(fields, values), nil
}

func (this *ModelStruct) ParseIDString(s string) (_ ID, err error) {
	if s == "" {
		return nil, errors.New("parse id from string: empty string")
	}
	if len(this.PrimaryFields) == 1 {
		var (
			id    = this.DefaultID()
			value = id.Value()
		)
		if value, err = value.ParseString(s); err != nil {
			return
		}
		return id.SetValue(value)
	}
	var b []byte
	if b, err = base64.RawURLEncoding.DecodeString(s); err != nil {
		return
	}
	return this.ParseIDBytes(b)
}
