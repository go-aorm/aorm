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
	/*
		if prefix := lcp.LCP([]byte(aName), []byte(bName)); len(prefix) > 0 {
			for i := len(prefix) - 1; i >= 1; i-- {
				if prefix[i] == '_' {
					prefix = prefix[0:i]
					break
				}
			}
			l := len(prefix)
			a, b := string(prefix), string(prefix) + "__" + strings.TrimPrefix(aName[l:], "_") + "__" + strings.TrimPrefix(bName[l:], "_")
			return a, b
		}*/
	return "", aName + "__" + bName
}
