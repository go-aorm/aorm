package gorm

import (
	_ "unsafe"
	"github.com/alangpierce/go-forceexport"
)

var convertAssign func (dest, src interface{}) error

func init() {
	err := forceexport.GetFunc(&convertAssign, "database/sql.convertAssign")
	if err != nil {
		panic("import database/sql.convertAssign failed: " + err.Error())
	}
}