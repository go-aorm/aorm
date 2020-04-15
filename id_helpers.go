package aorm

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
