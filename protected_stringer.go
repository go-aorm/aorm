package aorm

import (
	"database/sql/driver"
	"fmt"
)

const HiddenStringerValue = "«hidden value»"

type (
	ProtectedStringer interface {
		CanProtectStringer() bool
	}

	ProtectedStringerImpl struct{}

	ProtectedString string
)

func (ProtectedStringerImpl) CanProtectStringer() bool {
	return true
}

func (this *ProtectedString) Scan(src interface{}) error {
	if src == nil {
		*this = ""
	} else {
		switch t := src.(type) {
		case []byte:
			*this = ProtectedString(t)
		case string:
			*this = ProtectedString(t)
		default:
			return fmt.Errorf("aorm.ProtectedString.Scan(%T): unexpected type", src)
		}
	}
	return nil
}

func (ProtectedString) String() string {
	return HiddenStringerValue
}

func (this ProtectedString) Value() (driver.Value, error) {
	return string(this), nil
}

func (ProtectedString) CanProtectStringer() bool {
	return true
}
