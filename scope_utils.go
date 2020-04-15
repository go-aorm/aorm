package aorm

import "reflect"

func convertInterfaceToMap(scope *Scope, values interface{}, withIgnoredField, storeBlankField bool) map[string]interface{} {
	var (
		attrs = map[string]interface{}{}
	)

	switch value := values.(type) {
	case map[string]interface{}:
		return value
	case []interface{}:
		for _, v := range value {
			for key, value := range convertInterfaceToMap(scope, v, withIgnoredField, storeBlankField) {
				attrs[key] = value
			}
		}
	case interface{}:
		var (
			reflectValue = reflect.ValueOf(values)
		)

		switch reflectValue.Kind() {
		case reflect.Map:
			for _, key := range reflectValue.MapKeys() {
				attrs[ToDBName(key.Interface().(string))] = reflectValue.MapIndex(key).Interface()
			}
		default:
			var (
				acceptNames  = map[string]bool{}
				excludeNames = map[string]bool{}
				accept       = func(name string) bool {
					if _, ok := excludeNames[name]; ok {
						return false
					}
					return len(acceptNames) == 0 || acceptNames[name]
				}
			)

			switch scope.Operation {
			case OpUpdate:
				if i, ok := value.(FieldsForUpdateAcceptor); ok {
					if names, ok := i.AormAcceptFieldsForUpdate(scope); ok {
						for _, name := range names {
							acceptNames[name] = true
						}
					}
				}
				if i, ok := value.(FieldsForUpdateExcluder); ok {
					if names, ok := i.AormExcludeFieldsForUpdate(scope); ok {
						for _, name := range names {
							excludeNames[name] = true
						}
					}
				}
			case OpCreate:
				if i, ok := value.(FieldsForCreateAcceptor); ok {
					if names, ok := i.AormAcceptFieldsForCreate(scope); ok {
						for _, name := range names {
							acceptNames[name] = true
						}
					}
				}
				if i, ok := value.(FieldsForCreateAcceptor); ok {
					if names, ok := i.AormAcceptFieldsForCreate(scope); ok {
						for _, name := range names {
							excludeNames[name] = true
						}
					}
				}
			}

			for _, field := range InstanceOf(value).Fields {
				if (storeBlankField || !field.IsBlank) && (withIgnoredField || !field.IsIgnored) && field.Relationship == nil && accept(field.Name) {
					attrs[field.DBName] = field.Field.Interface()
				}
			}
		}
	}
	return attrs
}
