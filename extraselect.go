package aorm

import "reflect"

type extraSelect struct {
	Clause
	key    string
	Values []interface{}
	Ptrs   []bool
}

type extraSelects struct {
	Items     []*extraSelect
	Types     []reflect.Type
	Size      int
	Callbacks []func(recorde interface{}, data map[string]*ExtraResult)
}

func (es *extraSelects) NewValues() []interface{} {
	r := make([]interface{}, len(es.Types))
	for i, typ := range es.Types {
		r[i] = reflect.New(typ).Interface()
	}
	return r
}

func (es *extraSelects) Callback(f ...func(recorde interface{}, data map[string]*ExtraResult)) {
	es.Callbacks = append(es.Callbacks, f...)
}

func (es *extraSelects) Add(key string, values []interface{}, query interface{}, args []interface{}) *extraSelect {
	types := make([]reflect.Type, len(values))
	ptrs := make([]bool, len(values))
	for i, t := range values {
		elem := reflect.ValueOf(t)
		for elem.Kind() == reflect.Ptr {
			ptrs[i] = true
		}
		types[i] = reflect.Indirect(elem).Type()
	}
	s := &extraSelect{Clause{query, args}, key, values, ptrs}
	es.Types = append(es.Types, types...)
	es.Items = append(es.Items, s)
	es.Size = len(es.Types)
	return s
}

type ExtraResult struct {
	Select *extraSelect
	Values []interface{}
	Names  []string
	Map    map[string]int
}

func (er *ExtraResult) Get(name string) (v interface{}, ok bool) {
	var i int
	if i, ok = er.Map[name]; ok {
		v = er.Values[i]
	}
	return
}

type ExtraSelectModel struct {
	ExtraScannedValues map[string]*ExtraResult `sql:"-"`
}

func (es *ExtraSelectModel) SetAormExtraScannedValues(extra map[string]*ExtraResult) {
	es.ExtraScannedValues = extra
}

func (es *ExtraSelectModel) GetAormExtraScannedValue(name string) (result *ExtraResult, ok bool) {
	if es.ExtraScannedValues == nil {
		return
	}
	result, ok = es.ExtraScannedValues[name]
	return
}

func (es *ExtraSelectModel) GetAormExtraScannedValues() map[string]*ExtraResult {
	if es.ExtraScannedValues == nil {
		es.ExtraScannedValues = make(map[string]*ExtraResult)
	}
	return es.ExtraScannedValues
}
