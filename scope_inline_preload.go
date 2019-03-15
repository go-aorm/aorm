package aorm

import "reflect"

// AutoInlinePreload preload associations
func (s *Scope) AutoInlinePreload() *Scope {
	key := InlinePreloadFieldsKeyOf(s.Value)

	if data, ok := s.Get(key); ok {
		for k := range data.(map[string]bool) {
			s.Search.InlinePreload(k)
		}
		return s
	}

	modelStruct := s.GetModelStruct()

	value := reflect.New(modelStruct.ModelType).Interface()
	if ipf, ok := value.(InlinePreloadFields); ok {
		for _, fieldName := range ipf.GetGormInlinePreloadFields() {
			if f, ok := modelStruct.StructFieldsByName[fieldName]; ok {
				if f.Relationship != nil {
					s.Search.InlinePreload(f.Name)
				}
			}
		}
	}

	if modelStruct.virtualFieldsAutoInlinePreload != nil {
		for _, fieldName := range modelStruct.virtualFieldsAutoInlinePreload {
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
