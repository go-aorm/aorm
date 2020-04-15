package aorm

type (
	InlinePreloadFields interface {
		GetAormInlinePreloadFields() []string
	}

	InlinePreloadFieldsWithScope interface {
		GetAormInlinePreloadFields(scope *Scope) []string
	}

	AfterInlinePreloadScanner interface {
		AormAfterInlinePreloadScan(ip *InlinePreloader, recorde, value interface{})
	}
)
