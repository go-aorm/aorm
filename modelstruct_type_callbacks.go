package aorm

import (
	"strings"
	"sync"
)

type TypeCallbackData struct {
	ParentScope *Scope
	Scope       *Scope
	Model       *ModelStruct
	Path        []interface{}
}

type TypeCallback = func(data *TypeCallbackData)

type TypeCallbacksRegistrator struct {
	m  map[string]map[CallbackPosition][]TypeCallback
	mu sync.RWMutex
}

func (this *TypeCallbacksRegistrator) Register(pos CallbackPosition, name []string, f ...TypeCallback) {
	this.mu.Lock()
	defer this.mu.Unlock()

	if this.m == nil {
		this.m = map[string]map[CallbackPosition][]TypeCallback{}
	}

	for _, name := range name {
		if name == "" {
			continue
		}
		if callbacks, ok := this.m[name]; !ok {
			this.m[name] = map[CallbackPosition][]TypeCallback{pos: f}
		} else if _, ok = callbacks[pos]; !ok {
			callbacks[pos] = f
		} else {
			this.m[name][pos] = append(this.m[name][pos], f...)
		}
	}
}

func (this *TypeCallbacksRegistrator) Call(methodName string, pos CallbackPosition, scope, parentScope *Scope) (ok bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	return this.call(methodName, pos, TypeCallbackData{
		Model:       scope.modelStruct,
		Scope:       scope,
		ParentScope: scope,
	})
}

func (this *TypeCallbacksRegistrator) call(methodName string, pos CallbackPosition, data TypeCallbackData) (ok bool) {
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
		if !this.call(methodName, pos, childData) {
			return
		}
	}

	if len(data.Model.virtualFields) > 0 {
		for _, field := range data.Model.virtualFields {
			childData := data
			childData.Model = field.Model
			childData.Path = append(childData.Path, field.FieldName)
			if !this.call(methodName, pos, childData) {
				return
			}
		}
	}
	return true
}

type TypeCallbacks struct {
	TypeRegistrator TypeCallbacksRegistrator
}

func (this *TypeCallbacks) TypeCallback(pos CallbackPosition, name []string, f ...TypeCallback) *TypeCallbacks {
	this.TypeRegistrator.Register(pos, name, f...)
	return this
}

func (this *TypeCallbacks) TypeCallbackName(pos CallbackPosition, name string, f ...TypeCallback) *TypeCallbacks {
	return this.TypeCallback(pos, strings.Split(name, ","), f...)
}

func (this *TypeCallbacks) Migrate(pos CallbackPosition, f ...TypeCallback) *TypeCallbacks {
	return this.TypeCallbackName(pos, "Migrate", f...)
}

func (this *TypeCallbacks) CreateTable(pos CallbackPosition, f ...TypeCallback) *TypeCallbacks {
	return this.TypeCallbackName(pos, "CreateTable", f...)
}
