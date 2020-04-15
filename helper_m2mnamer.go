package aorm

import (
	"reflect"
	"unicode"

	"github.com/abcum/lcp"
)

func M2MNameOf(a reflect.Type, b reflect.Type) (prefix, name string) {
	return M2MNameOfString(indirectRealType(a).Name(), indirectRealType(b).Name())
}

func M2MNameOfString(aName, bName string) (prefix, name string) {
	if prefix := lcp.LCP([]byte(aName), []byte(bName)); len(prefix) > 0 {
		for i := len(prefix) - 1; i >= 1; i-- {
			if unicode.IsUpper(rune(prefix[i])) {
				prefix = prefix[0:i]
				break
			}
		}
		l := len(prefix)
		return string(prefix), ToDBName(string(prefix)) + "__" + ToDBName(aName[l:]) + "__" + ToDBName(bName[l:])
	}
	return "", ToDBName(aName) + "__" + ToDBName(bName)
}
