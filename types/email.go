package types

import (
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/moisespsena-go/aorm"
)

const EmailSize = uint16(255)

type (
	Email string

	EmailAssigner struct {
	}
)

func init() {
	aorm.Register(EmailAssigner{})
}

func (this Email) PrimaryGoValue() interface{} {
	return string(this)
}

func (this Email) IsZero() bool {
	return this == ""
}

func (EmailAssigner) Valuer(_ aorm.Dialector, value interface{}) driver.Valuer {
	return aorm.ValuerFunc(func() (driver.Value, error) {
		return string(value.(Email)), nil
	})
}

func (EmailAssigner) Scaner(_ aorm.Dialector, dest reflect.Value) aorm.Scanner {
	return aorm.ScannerFunc(func(src interface{}) (err error) {
		if src == nil {
			dest.SetString("")
			return
		}
		dest.SetString(src.(string))
		return
	})
}

func (EmailAssigner) SQLType(aorm.Dialector) string {
	return fmt.Sprintf("VARCHAR(%d)", EmailSize)
}

func (EmailAssigner) SQLSize(_ aorm.Dialector) int {
	return int(EmailSize)
}

func (EmailAssigner) Type() reflect.Type {
	return reflect.TypeOf(Email(0))
}
