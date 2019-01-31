package assigners

import (
	"database/sql/driver"
	"encoding/json"
	"reflect"

	"github.com/moisespsena-go/aorm"
)

func init() {
	Register(&Strings{})
}

type Strings struct{}

func (Strings) Valuer(dialect aorm.Dialect, value interface{}) driver.Valuer {
	return aorm.ValuerFunc(func() (v driver.Value, err error) {
		slice := value.([]string)
		if slice == nil {
			return
		}
		v, err = json.Marshal(slice)
		return v, err
	})
}

func (Strings) Scaner(dialect aorm.Dialect, dest reflect.Value) aorm.Scanner {
	return &aorm.ScannerFunc{
		Func: func(src interface{}) (err error) {
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
		}}
}

func (Strings) SQLType(dialect aorm.Dialect) string {
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
