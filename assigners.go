package aorm

var assigners = &AssignerRegistrator{}

func Assigners() *AssignerRegistrator {
	return assigners
}

func Register(assigner ...Assigner) {
	assigners.Register(assigner...)
}
