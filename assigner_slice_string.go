package aorm

import (
	"database/sql/driver"
	"encoding/json"
	"reflect"
)

func init() {
	Register(&Strings{})
}

type Strings struct{}

func (Strings) Valuer(_ Dialector, value interface{}) driver.Valuer {
	return ValuerFunc(func() (v driver.Value, err error) {
		slice := value.([]string)
		if slice == nil {
			return
		}
		v, err = json.Marshal(slice)
		return v, err
	})
}

func (Strings) Scaner(_ Dialector, dest reflect.Value) Scanner {
	return ScannerFunc(func(src interface{}) (err error) {
		var slice []string
		if src != nil {
			var data []byte
			switch t := src.(type) {
			case string:
				data = []byte(t)
			case []byte:
				data = t
			}
			if err = json.Unmarshal(data, &slice); err != nil {
				return
			}
		}
		dest.Set(reflect.ValueOf(slice))
		return
	})
}

func (Strings) SQLType(dialect Dialector) string {
	switch dialect.GetName() {
	case "postgres":
		return "jsonb"
	default:
		return "text"
	}
}

func (Strings) Type() reflect.Type {
	return reflect.TypeOf([]string{})
}
