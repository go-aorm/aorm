package aorm

import (
	"strings"
)

func JoinNameOfString(aName, bName string) (prefix, name string) {
	a := strings.Split(aName, "_")
	if len(a) > 1 {
		if strings.HasPrefix(bName, a[0]+"_") {
			return a[0], aName + "__" + strings.TrimPrefix(bName, a[0]+"_")
		}
	}
	return "", aName + "__" + bName
}
