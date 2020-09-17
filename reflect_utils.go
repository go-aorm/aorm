package aorm

import (
	"go/ast"
	"reflect"
)

type IndexableStructField struct {
	reflect.StructField
	Index []int
}

func getTypeFields(typ interface{}) (result []IndexableStructField) {
	var (
		walk    func(typ reflect.Type, path []int)
		nameMap = map[string]interface{}{}
		fields  []IndexableStructField
	)
	walk = func(typ reflect.Type, path []int) {
		typ = indirectType(typ)
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if !ast.IsExported(field.Name) {
				if field.Anonymous {
					walk(field.Type, append(path, i))
				}
				continue
			}
			fields = append(fields, IndexableStructField{field, append(path, field.Index...)})
		}
	}
	switch t := typ.(type) {
	case reflect.Type:
		walk(t, nil)
	default:
		walk(reflect.TypeOf(typ), nil)
	}

	for i := len(fields) - 1; i >= 0; i-- {
		f := fields[i]
		if _, ok := nameMap[f.Name]; !ok {
			nameMap[f.Name] = nil
			result = append(result, f)
		}
	}
	for left, right := 0, len(result)-1; left < right; left, right = left+1, right-1 {
		result[left], result[right] = result[right], result[left]
	}
	return
}
