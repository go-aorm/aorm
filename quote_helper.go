package aorm

import (
	"regexp"
	"strings"
)

var columnRegexp = regexp.MustCompile("^[a-zA-Z\\d]+(\\.[a-zA-Z\\d]+)*$") // only match string like `name`, `users.name`

// QuotePath used to quote string to escape them for database
func QuotePath(q Quoter, str string) string {
	if strings.Index(str, ".") != -1 {
		newStrs := []string{}
		for _, str := range strings.Split(str, ".") {
			newStrs = append(newStrs, Quote(q, str))
		}
		return strings.Join(newStrs, ".")
	}

	return Quote(q, str)
}

func QuoteIfPossible(q Quoter, str string) string {
	if columnRegexp.MatchString(str) {
		return QuotePath(q, str)
	}
	return str
}
