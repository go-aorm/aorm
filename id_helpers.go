package aorm

import (
	"errors"
	"fmt"
)

func InID(id ...ID) *idSliceScoper {
	return &idSliceScoper{values: id}
}

func InIDNamedTable(tableName string, id ...ID) *idSliceNamedTable {
	return &idSliceNamedTable{TableName: tableName, values: id}
}

func IDSlice(args ...interface{}) (r []ID) {
	for _, arg := range args {
		switch t := arg.(type) {
		case string:
			if t != "" {
				r = append(r, FakeID(t))
			}
		case []string:
			for _, v := range t {
				r = append(r, FakeID(v))
			}
		/*case int64:
			r = append(r, IntID(uint64(t)))
		case []int64:
			for _, v := range t {
				r = append(r, IntID(uint64(v)))
			}
		case uint64:
			r = append(r, IntID(t))
		case []uint64:
			for _, v := range t {
				r = append(r, IntID(v))
			}*/
		case ID:
			r = append(r, t)
		case []ID:
			r = append(r, t...)
		}
	}
	return
}

func FieldOfId(id ID, name ...string) *StructField {
	if len(name) == 0 {
		if len(id.Values()) > 1 {
			panic(errors.New("require name"))
		}
		return id.Fields()[0]
	}
	for i, f := range id.Fields() {
		if f.Name == name[0] {
			return id.Fields()[i]
		}
	}
	panic(fmt.Errorf("primary field %q doesn't exists", name[0]))
}

func IDRawToMap(id ID) map[string]interface{} {
	m := map[string]interface{}{}
	raws := RawsOfId(id)
	for i, f := range id.Fields() {
		m[f.Name] = raws[i]
	}
	return m
}

func ValueOfId(id ID, name ...string) IDValuer {
	if len(name) == 0 {
		if len(id.Values()) > 1 {
			panic(errors.New("require name"))
		}
		return id.Values()[0]
	}
	for i, f := range id.Fields() {
		if f.Name == name[0] {
			return id.Values()[i]
		}
	}
	panic(fmt.Errorf("primary field %q doesn't exists", name[0]))
}

func RawOfId(id ID, name ...string) interface{} {
	return ValueOfId(id, name...).Raw()
}

func RawsOfId(id ID, name ...string) (raws []interface{}) {
	if len(name) == 0 {
		for _, v := range id.Values() {
			raws = append(raws, v.Raw())
		}
		return
	}
	for i, f := range id.Fields() {
		for _, name := range name {
			if f.Name == name {
				raws = append(raws, id.Values()[i].Raw())
			}
		}
	}
	return
}

func IDOf(value interface{}) ID {
	return StructOf(value).GetID(value)
}

func CopyIdTo(src, dst ID) (ID, error) {
	var (
		srcValues = src.Values()
		values    = make([]interface{}, len(srcValues))
	)

	for i, sf := range src.Fields() {
		for j, df := range dst.Fields() {
			if df.Name == sf.Name {
				values[j] = srcValues[i]
			}
		}
	}

	return dst.SetValue(values...)
}

func SetIDValuersToRecord(model *ModelStruct, record interface{}, valuer ...interface{}) (err error) {
	if id, err := model.DefaultID().SetValue(valuer...); err != nil {
		return err
	} else {
		id.SetTo(record)
		return nil
	}
}
