package aorm

import (
	"path"

	path_helpers "github.com/moisespsena-go/path-helpers"
)

type pkgNameRegister struct {
	m safeMap
}

func (this *pkgNameRegister) Set(pkgPath, prefix string) {
	this.m.Set(pkgPath, prefix)
}

func (this *pkgNameRegister) Get(pkgPath string) (prefix string) {
	return this.m.Get(pkgPath)
}

func (this *pkgNameRegister) Auto(sub ...string) {
	pkgPath := path_helpers.GetCalledDirUp(1)
	this.m.Set(pkgPath, path.Join(append([]string{pkgPath}, sub...)...))
}

func (this *pkgNameRegister) Parent() {
	this.Parents(1)
}

func (this *pkgNameRegister) Parents(up int) {
	pkgPath := path_helpers.GetCalledDirUp(up + 1)
	pth := pkgPath
	for i := 0; i < up; i++ {
		pth = path.Dir(pth)
	}
	this.m.Set(pkgPath, pth)
}

var PkgNamer pkgNameRegister
