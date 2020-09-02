package aorm

import (
	"reflect"
	"strings"
	"sync"
)

const (
	Before CallbackPosition = iota
	After
)

type ScopeCallbackData struct {
	Scope       *Scope
	Model       *ModelStruct
	RecordValue reflect.Value
	Path        []interface{}
}

func (this *ScopeCallbackData) Record() interface{} {
	return this.RecordValue.Interface()
}

type ScopeCallback = func(data *ScopeCallbackData)
type CallbackPosition uint8

type ScopeCallbacksRegistrator struct {
	m  map[string]map[CallbackPosition][]ScopeCallback
	mu sync.RWMutex
}

func (this *ScopeCallbacksRegistrator) Register(pos CallbackPosition, name []string, f ...ScopeCallback) {
	this.mu.Lock()
	defer this.mu.Unlock()

	if this.m == nil {
		this.m = map[string]map[CallbackPosition][]ScopeCallback{}
	}

	for _, name := range name {
		if name == "" {
			continue
		}
		if callbacks, ok := this.m[name]; !ok {
			this.m[name] = map[CallbackPosition][]ScopeCallback{pos: f}
		} else if _, ok = callbacks[pos]; !ok {
			callbacks[pos] = f
		} else {
			this.m[name][pos] = append(this.m[name][pos], f...)
		}
	}
}

func (this *ScopeCallbacksRegistrator) Call(methodName string, pos CallbackPosition, scope *Scope, recordValue reflect.Value) (ok bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.call(methodName, pos, ScopeCallbackData{
		Model:       scope.modelStruct,
		Scope:       scope,
		RecordValue: recordValue,
	})
}

func (this *ScopeCallbacksRegistrator) call(methodName string, pos CallbackPosition, data ScopeCallbackData) (ok bool) {
	if positions, ok2 := this.m[methodName]; ok2 {
		if callbacks, ok2 := positions[pos]; ok2 {
			for _, cb := range callbacks {
				cb(&data)
				if data.Scope.HasError() {
					return
				}
			}
		}
	}

	for _, child := range data.Model.Children {
		field := data.Model.FieldsByName[child.ParentField.Name]
		childData := data
		childData.Model = field.Model
		childData.Path = append(childData.Path, childData.Model.ParentField.Name)
		childData.RecordValue = reflect.Indirect(data.RecordValue).FieldByIndex(field.StructIndex)
		if childData.RecordValue.Kind() != reflect.Ptr {
			childData.RecordValue = childData.RecordValue.Addr()
		}
		if !this.callChild(methodName, pos, childData) {
			return
		}
	}

	if len(data.Model.virtualFields) > 0 {
		record := data.Record()
		for _, field := range data.Model.virtualFields {
			if value, ok2 := field.Get(record); ok2 {
				childData := data
				childData.Model = field.Model
				childData.Path = append(childData.Path, field.FieldName)
				childData.RecordValue = reflect.ValueOf(value)
				if childData.RecordValue.Kind() != reflect.Ptr {
					childData.RecordValue = childData.RecordValue.Addr()
				}
				if !this.callChild(methodName, pos, childData) {
					return
				}
			}
		}
	}
	return true
}

func (this *ScopeCallbacksRegistrator) callChild(methodName string, pos CallbackPosition, data ScopeCallbackData) (ok bool) {
	childValue := data.RecordValue
retry:
	k := childValue.Kind()
	switch k {
	case reflect.Ptr:
		if childValue.IsNil() {
			return true
		}
		childValue = childValue.Elem()
		goto retry
	case reflect.Slice:
		ptr := childValue.Type().Elem().Kind() == reflect.Ptr
		childData := data
		childData.Path = append(childData.Path, 0)
		x := len(childData.Path) - 1
		for i, l := 0, childValue.Len(); i < l; i++ {
			childData.Path[x] = i
			childData.RecordValue = childValue.Index(i)
			if !ptr {
				childData.RecordValue = childData.RecordValue.Addr()
			}
			if !data.Model.ScopeCallbacks.Registrator.call(methodName, pos, childData) {
				return
			}
		}
	case reflect.Struct:
		childData := data
		childData.RecordValue = childValue.Addr()
		if !data.Model.ScopeCallbacks.Registrator.call(methodName, pos, childData) {
			return
		}
	default:
		log.Warningf("scope callback for children kind=%s not implemented", k)
	}
	return true
}

type ScopeCallbacks struct {
	Registrator ScopeCallbacksRegistrator
}

func (this *ScopeCallbacks) ScopeCallback(pos CallbackPosition, name []string, f ...ScopeCallback) *ScopeCallbacks {
	this.Registrator.Register(pos, name, f...)
	return this
}

func (this *ScopeCallbacks) ScopeCallbackName(pos CallbackPosition, name string, f ...ScopeCallback) *ScopeCallbacks {
	return this.ScopeCallback(pos, strings.Split(name, ","), f...)
}
func (this *ScopeCallbacks) BeforeCreate(pos CallbackPosition, f ...ScopeCallback) *ScopeCallbacks {
	return this.ScopeCallbackName(pos, "BeforeCreate", f...)
}
func (this *ScopeCallbacks) AfterCreate(pos CallbackPosition, f ...ScopeCallback) *ScopeCallbacks {
	return this.ScopeCallbackName(pos, "AfterCreate", f...)
}
func (this *ScopeCallbacks) AfterSave(pos CallbackPosition, f ...ScopeCallback) *ScopeCallbacks {
	return this.ScopeCallbackName(pos, "AfterSave", f...)
}
func (this *ScopeCallbacks) AfterDelete(pos CallbackPosition, f ...ScopeCallback) *ScopeCallbacks {
	return this.ScopeCallbackName(pos, "AfterDelete", f...)
}
func (this *ScopeCallbacks) BeforeDelete(pos CallbackPosition, f ...ScopeCallback) *ScopeCallbacks {
	return this.ScopeCallbackName(pos, "BeforeDelete", f...)
}
func (this *ScopeCallbacks) AfterFind(pos CallbackPosition, f ...ScopeCallback) *ScopeCallbacks {
	return this.ScopeCallbackName(pos, "AfterFind", f...)
}
func (this *ScopeCallbacks) AfterUpdate(pos CallbackPosition, f ...ScopeCallback) *ScopeCallbacks {
	return this.ScopeCallbackName(pos, "AfterUpdate", f...)
}
func (this *ScopeCallbacks) BeforeSave(pos CallbackPosition, f ...ScopeCallback) *ScopeCallbacks {
	return this.ScopeCallbackName(pos, "BeforeSave", f...)
}
func (this *ScopeCallbacks) BeforeUpdate(pos CallbackPosition, f ...ScopeCallback) *ScopeCallbacks {
	return this.ScopeCallbackName(pos, "BeforeUpdate", f...)
}
func (this *ScopeCallbacks) AfterScan(pos CallbackPosition, f ...ScopeCallback) *ScopeCallbacks {
	return this.ScopeCallbackName(pos, "AfterScan", f...)
}
