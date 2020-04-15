package aorm

import (
	"database/sql/driver"
	"encoding/base64"
	"encoding/hex"
	"reflect"

	"github.com/moisespsena-go/bid"
)

func init() {
	Register(BIDAssigner{})
}

type BIDAssigner struct {
}

func (a BIDAssigner) DbBindVar(dialect Dialector, argVar string) string {
	switch dialect.GetName() {
	case "postgres":
		return argVar + "::BYTEA"
	}
	return argVar
}

func (BIDAssigner) Valuer(dialect Dialector, value interface{}) driver.Valuer {
	d := dialect.GetName()
	switch d {
	case "postgres":
		return ValuerFunc(func() (r driver.Value, err error) {
			var v []byte
			switch t := value.(type) {
			case bid.BID:
				if t.IsZero() {
					return
				}
				v = t[:]
			case string:
				v = []byte(t)
			case []byte:
				v = t
			default:
				return
			}
			return "\\x" + hex.EncodeToString(v), nil
		})
	default:
		return ValuerFunc(func() (driver.Value, error) {
			return value.(bid.BID).AsBytes(), nil
		})
	}
}

func (BIDAssigner) Scaner(_ Dialector, dest reflect.Value) Scanner {
	return ScannerFunc(func(src interface{}) (err error) {
		var value bid.BID
		if err = value.Scan(src); err == nil {
			dest.Set(reflect.ValueOf(value))
		}
		return
	})
}

func (BIDAssigner) SQLType(dialect Dialector) string {
	switch dialect.GetName() {
	case "postgres":
		return "BYTEA"
	case "sqlite", "sqlite3":
		return "BLOB"
	}
	return "CHAR(12)"
}

func (BIDAssigner) SQLSize(_ Dialector) int {
	return 0
}

func (BIDAssigner) Type() reflect.Type {
	return bidType
}

func (BIDAssigner) FromRaw(raw interface{}) IDValuer {
	return BIDIdValuer(raw.(bid.BID))
}

func (BIDAssigner) ToRaw(value IDValuer) interface{} {
	return bid.BID(value.(BytesId))
}

type BIDIdValuer BytesId

func (this BIDIdValuer) Bytes() []byte {
	return this
}

func (BIDIdValuer) ParseBytes(b []byte) (IDValuer, error) {
	vlr := make([]byte, len(b))
	copy(vlr, b)
	return BIDIdValuer(vlr), nil
}

func (BIDIdValuer) ParseString(s string) (_ IDValuer, err error) {
	var b BIDIdValuer
	b, err = base64.RawURLEncoding.DecodeString(s)
	return b, nil
}

func (this BIDIdValuer) String() string {
	return base64.RawURLEncoding.EncodeToString(this)
}

func (this BIDIdValuer) IsZero() bool {
	return len(this) == 0
}

func (this BIDIdValuer) Raw() interface{} {
	return bid.BID(this)
}
