package aorm

import (
	"path/filepath"
	"reflect"
	"strings"
)

func TableNameOfPrefix(prefix, name string, names ...string) []string {
	names = append(names, "")
	copy(names[1:], names)
	names[0] = name

	if prefix != "" && prefix != "main" {
		for i, name := range names {
			if name == prefix || strings.HasPrefix(name, prefix) {
				continue
			} else if name[0] == '.' {
				names[i] = prefix + "_" + name[1:]
			} else {
				names[i] = prefix + "_" + name
			}
		}
	}

	return names
}

func TableNameOf(pkgPath string, name string, names ...string) []string {
	return TableNameOfPrefix(TableNamePrefixOf(pkgPath), name, names...)
}

func TableNamePrefixOf(pkgPath string, defaul ...string) (prefix string) {
	defer func() {
		if prefix == "" {
			for _, prefix = range defaul {
			}
		}
	}()

	prefix = TableNamePrefixRegister.Get(pkgPath)
	if prefix != "" {
		return
	}

	pkgPath = strings.TrimSuffix(pkgPath, "/models")

	// if main pkg and prefix for it isn't registered, skip
	if pkgPath == "" || pkgPath == "models" {
		return
	}
	return ToDBName(NamifyString(filepath.Base(pkgPath)))
}

func TableNamePrefixOfInterface(value interface{}) (prefix string) {
	if typ, ok := value.(reflect.Type); ok {
		value = reflect.New(indirectRealType(typ)).Interface()
	}
	if prefixer, ok := value.(TableNamePrefixer); ok {
		return prefixer.TableNamePrefix()
	}
	return TableNamePrefixOf(indirectRealType(reflect.TypeOf(value)).PkgPath())
}

func TableNamesOfInterface(value interface{}, name string, names ...string) []string {
	return TableNameOfPrefix(TableNamePrefixOfInterface(value), name, names...)
}
