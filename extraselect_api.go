package aorm

type ExtraSelectInterface interface {
	SetAormExtraScannedValues(extra map[string]*ExtraResult)
	GetAormExtraScannedValue(name string) (result *ExtraResult, ok bool)
	GetAormExtraScannedValues() map[string]*ExtraResult
}
