package aorm

type (
	Zeroer interface {
		IsZero() bool
	}

	Reseter interface {
		Reset()
	}
)
