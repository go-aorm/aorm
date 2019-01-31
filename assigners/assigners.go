package assigners

import (
	"github.com/moisespsena-go/aorm"
)

var assigners = &aorm.AssignerRegistrator{}

func Assigners() *aorm.AssignerRegistrator {
	return assigners
}

func Register(assigner ...aorm.Assigner) {
	assigners.Register(assigner...)
}
