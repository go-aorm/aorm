package aorm

import "strings"

const (
	DefaultQuoter = QuoteRuner('`')
	QuoteCharS    = string(DefaultQuoter)
)

type (
	Quoter interface {
		// DefaultQuoter quotes char for field name to avoid SQL parsing exceptions by using a reserved word as a field name
		QuoteChar() rune
	}

	QuoteRuner rune
)

func (this QuoteRuner) QuoteChar() rune {
	return rune(this)
}

// Quote quotes field name to avoid SQL parsing exceptions by using a reserved word as a field name
func Quote(q Quoter, key string) string {
	qc := string(q.QuoteChar())
	return qc + key + qc
}

// QuoteConvertChar converts all characters equals fromChar in string to quote char
func QuoteConvertChar(q Quoter, fromChar string, s string) string {
	qc := string(q.QuoteChar())
	if qc == fromChar {
		return s
	}
	return strings.ReplaceAll(s, fromChar, qc)
}

// QuoteConvert converts all characters equals QuoteChar in string to quote char
func QuoteConvert(q Quoter, s string) string {
	return QuoteConvertChar(q, QuoteCharS, s)
}
