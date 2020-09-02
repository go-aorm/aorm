package types

import (
	"bytes"
	"compress/gzip"
	"database/sql/driver"
	"fmt"
	"io/ioutil"
	"reflect"

	"github.com/moisespsena-go/aorm"
)

type (
	Gzb struct {
		Compressed bool
		Data       []byte
	}

	GzbAssigner struct {
	}
)

func (this Gzb) Value() (driver.Value, error) {
	var out = make([]byte, len(this.Data)+1, len(this.Data)+1)
	if this.Compressed {
		out[1] = 1
		copy(out[1:], this.Data)
	} else if len(this.Data) > 512 {
		var w bytes.Buffer
		w.WriteByte(1)
		gzw := gzip.NewWriter(&w)
		if _, err := gzw.Write(this.Data); err != nil {
			return "", err
		}
		gzw.Close()
		return w.Bytes(), nil
	} else {
		copy(out[1:], this.Data)
	}
	return out, nil
}

func init() {
	aorm.Register(GzbAssigner{})
}

func (this *Gzb) Scan(src interface{}) error {
	*this = Gzb{}
	if src == nil || len(src.([]byte)) == 0 {
		return nil
	}
	b := src.([]byte)
	this.Compressed = b[0] == 1
	this.Data = b[1:]
	return nil
}

func (this Gzb) IsZero() bool {
	return len(this.Data) == 0
}

func (this *Gzb) MustGet() (b []byte) {
	b, _ = this.Get()
	return
}

func (this *Gzb) String() string {
	return string(this.MustGet())
}

func (this *Gzb) Get() (_ []byte, err error) {
	if this.Compressed {
		buf := bytes.NewBuffer(this.Data)
		var r *gzip.Reader
		if r, err = gzip.NewReader(buf); err != nil {
			return
		}
		var uc []byte
		if uc, err = ioutil.ReadAll(r); err != nil {
			return
		}
		this.Compressed = false
		this.Data = uc
		return uc, nil
	}
	return this.Data, nil
}

func (GzbAssigner) Valuer(_ aorm.Dialector, value interface{}) driver.Valuer {
	return value.(Gzb)
}

func (GzbAssigner) Scaner(_ aorm.Dialector, dest reflect.Value) aorm.Scanner {
	return dest.Addr().Interface().(*Gzb)
}

func (GzbAssigner) SQLType(d aorm.Dialector) string {
	if d.GetName() == "postgres" {
		return "BYTEA"
	}
	return fmt.Sprintf("BLOB")
}

func (GzbAssigner) SQLSize(_ aorm.Dialector) int {
	return 0
}

func (GzbAssigner) Type() reflect.Type {
	return reflect.TypeOf(Gzb{})
}
