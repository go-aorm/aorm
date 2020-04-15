package aorm

import (
	"fmt"
	"reflect"
)

// AutoInlinePreload preload associations
func (s *Scope) AutoInlinePreload() *Scope {
	if s.Value == nil {
		return s
	}
	if _, skip := s.InstanceGet(OptSkipPreload); skip {
		return s
	}
	if _, skip := s.Get(OptSkipPreload); skip {
		return s
	}

	key := InlinePreloadFieldsKeyOf(s.Value)

	if data, ok := s.Get(key); ok {
		for k := range data.(map[string]bool) {
			s.Search.InlinePreload(k)
		}
		return s
	}

	modelStruct := s.Struct()

	value := reflect.New(modelStruct.Type).Interface()
	if ipf, ok := value.(InlinePreloadFields); ok {
		for _, fieldName := range ipf.GetAormInlinePreloadFields() {
			if f, ok := modelStruct.FieldsByName[fieldName]; ok {
				if f.Relationship != nil {
					s.Search.InlinePreload(f.Name)
				}
			} else if fieldName != "*" {
				panic(fmt.Errorf("Field %s#%s does not exists", modelStruct.Fqn(), fieldName))
			}
		}
	} else if ipf, ok := value.(InlinePreloadFieldsWithScope); ok {
		for _, fieldName := range ipf.GetAormInlinePreloadFields(s) {
			if f, ok := modelStruct.FieldsByName[fieldName]; ok {
				if f.Relationship != nil {
					s.Search.InlinePreload(f.Name)
				}
			} else if fieldName != "*" {
				panic(fmt.Errorf("Field %s#%s does not exists", modelStruct.Fqn(), fieldName))
			}
		}
	}

	if modelStruct.virtualFieldsAutoInlinePreload != nil {
	vfloop:
		for _, fieldName := range modelStruct.virtualFieldsAutoInlinePreload {
			for _, prl := range s.Search.inlinePreload {
				if prl.schema == fieldName {
					continue vfloop
				}
			}
			s.Search.InlinePreload(fieldName)
		}
	}

	return s
}

// InlinePreloadFields set inline preload fields of value type
func (s *Scope) InlinePreloadFields(value interface{}, fields ...string) *Scope {
	key := InlinePreloadFieldsKeyOf(value)
	new := map[string]bool{}

	if old, ok := s.Get(key); ok {
		for k := range old.(map[string]bool) {
			new[k] = true
		}
	}

	for _, f := range fields {
		if f[0] == '-' {
			f := f[1:]
			if _, ok := new[f]; ok {
				delete(new, f)
				continue
			}
		}
		new[f] = true
	}
	return s.Set(key, new)
}
