package aorm

import (
	"path"

	path_helpers "github.com/moisespsena-go/path-helpers"
)

type tablenamePrefixRegister struct {
	m safeMap
}

func (this *tablenamePrefixRegister) Set(pkgPath, prefix string) {
	this.m.Set(pkgPath, prefix)
}

func (this *tablenamePrefixRegister) Get(pkgPath string) (prefix string) {
	if prefix = this.m.Get(pkgPath); prefix == "" {
		if prefix = PkgNamer.Get(pkgPath); prefix != "" {
			prefix = path.Base(prefix)
		}
	}
	return
}

func (this *tablenamePrefixRegister) SetAuto(prefix string) {
	pkgPath := path_helpers.GetCalledDirUp(1)
	this.m.Set(pkgPath, prefix)
}

var TableNamePrefixRegister tablenamePrefixRegister
