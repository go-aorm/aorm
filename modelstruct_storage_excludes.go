package aorm

import (
	"reflect"
	"sync"
	"time"
)

var excludeTypesForModelStruct sync.Map

func init() {
	ExcludeTypeForModelStruct(
		time.Time{},
		time.Location{},
	)
}

func ExcludeTypeForModelStruct(typ ...interface{}) {
	for _, typ := range typ {
		var rtyp reflect.Type
		switch t := typ.(type) {
		case reflect.Type:
			rtyp = t
		case reflect.Value:
			rtyp = t.Type()
		default:
			rtyp = reflect.TypeOf(t)
		}

		if rtyp, _, _ = StructTypeOf(rtyp); rtyp != nil {
			excludeTypesForModelStruct.Store(rtyp, true)
		}
	}
}

func AcceptTypeForModelStruct(typ interface{}) (ok bool) {
	var rtyp reflect.Type
	switch t := typ.(type) {
	case reflect.Type:
		rtyp = t
	case reflect.Value:
		rtyp = t.Type()
	default:
		rtyp = reflect.TypeOf(t)
	}

	if rtyp, _, _ = StructTypeOf(rtyp); rtyp != nil {
		if value, _ := excludeTypesForModelStruct.Load(rtyp); value == nil {
			return true
		}
	}
	return
}

func AcceptableTypeForModelStructInterface(i interface{}) (typ reflect.Type) {
	if typ, _, _ = StructTypeOfInterface(i); typ != nil {
		if value, _ := excludeTypesForModelStruct.Load(typ); value == nil {
			return
		}
	}
	return nil
}
