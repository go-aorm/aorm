package aorm

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// preloadCallback used to preload associations
func preloadCallback(scope *Scope) {
	if scope.InstanceGetBool(OptKeySkipPreload) {
		return
	}
	if scope.GetBool(OptKeySkipPreload) {
		return
	}

	if scope.GetBool(OptKeyAutoPreload) {
		autoPreload(scope)
	}

	if scope.Search.preload == nil || scope.HasError() {
		return
	}

	var (
		preloadedMap = map[string]bool{}
		fields       = scope.Instance()
		oldes        = scope.Search.extraSelects
		oldesf       = scope.Search.extraSelectsFields
	)

	scope.Search.extraSelects = nil
	scope.Search.extraSelectsFields = nil

	defer func() {
		scope.Search.extraSelects = oldes
		scope.Search.extraSelectsFields = oldesf
	}()

	for _, preload := range scope.Search.preload {
		var (
			preloadFields = strings.Split(preload.schema, ".")
			currentScope  = scope
			currentFields = fields
		)

		for idx, preloadField := range preloadFields {
			var currentOptions *InlinePreloadOptions

			if currentScope == nil {
				continue
			}

			// if not preloaded
			if preloadKey := strings.Join(preloadFields[:idx+1], "."); !preloadedMap[preloadKey] {

				// assign search conditions to last preload
				if idx == len(preloadFields)-1 {
					currentOptions = preload.options
				}

				for _, field := range currentFields.Fields {
					if field.Name != preloadField || field.Relationship == nil {
						continue
					}

					switch field.Relationship.Kind {
					case "has_one":
						currentScope.handleHasOnePreload(field, &currentOptions.Conditions)
					case "has_many":
						currentScope.handleHasManyPreload(field, &currentOptions.Conditions)
					case "belongs_to":
						currentScope.handleBelongsToPreload(field, &currentOptions.Conditions)
					case "many_to_many":
						currentScope.handleManyToManyPreload(field, &currentOptions.Conditions)
					default:
						scope.Err(errors.New("unsupported relation"))
					}

					preloadedMap[preloadKey] = true
					break
				}

				if !preloadedMap[preloadKey] {
					scope.Err(fmt.Errorf("can'T preload field %s for %s", preloadField, currentScope.Struct().Type))
					return
				}
			}

			// preload next level
			if idx < len(preloadFields)-1 {
				currentScope = currentScope.ScopeOfField(preloadField)
				if currentScope != nil {
					currentFields = currentScope.Instance()
				}
			}
		}
	}
}

func autoPreload(scope *Scope) {
	for _, field := range scope.Instance().Fields {
		if field.Relationship == nil {
			continue
		}

		if val, ok := field.TagSettings["PRELOAD"]; ok {
			if preload, err := strconv.ParseBool(val); err != nil {
				scope.Err(errors.New("invalid preload option"))
				return
			} else if !preload {
				continue
			}
		}

		scope.Search.Preload(field.Name)
	}
}

func (scope *Scope) generatePreloadDBWithConditions(conditions []interface{}) (*DB, []interface{}) {
	var (
		preloadDB         = scope.NewDB()
		preloadConditions []interface{}
	)

	for _, condition := range conditions {
		if scopes, ok := condition.(func(*DB) *DB); ok {
			preloadDB = scopes(preloadDB)
		} else {
			preloadConditions = append(preloadConditions, condition)
		}
	}

	return preloadDB, preloadConditions
}

// handleHasOnePreload used to preload has one associations
func (scope *Scope) handleHasOnePreload(field *Field, conditions *Conditions) {
	relation := field.Relationship

	// get relations's primary keys
	primaryKeys := scope.getColumnAsArray(relation.AssociationForeignFieldNames, scope.Value)
	if len(primaryKeys) == 0 {
		return
	}

	// preload conditions
	preloadDB := conditions.MergeTo(scope.NewDB())

	// find relations
	query := fmt.Sprintf("%v IN (%v)", toQueryCondition(scope.db.dialect, relation.ForeignDBNames), toQueryMarks(primaryKeys))
	values := toQueryValues(primaryKeys)
	if relation.PolymorphicType != "" {
		query += fmt.Sprintf(" AND %v = ?", scope.Quote(relation.PolymorphicDBName))
		values = append(values, relation.PolymorphicValue)
	}

	results := makeSlice(field.Struct.Type)
	scope.Err(preloadDB.Where(query, values...).Find(results).Error)

	// assign find results
	var (
		resultsValue       = indirect(reflect.ValueOf(results))
		indirectScopeValue = scope.IndirectValue()
	)

	if indirectScopeValue.Kind() == reflect.Slice {
		for j := 0; j < indirectScopeValue.Len(); j++ {
			for i := 0; i < resultsValue.Len(); i++ {
				result := resultsValue.Index(i)
				foreignValues := getValueFromFields(result, relation.ForeignFieldNames)
				if indirectValue := indirect(indirectScopeValue.Index(j)); equalAsString(getValueFromFields(indirectValue, relation.AssociationForeignFieldNames), foreignValues) {
					indirectValue.FieldByName(field.Name).Set(result)
					break
				}
			}
		}
	} else {
		for i := 0; i < resultsValue.Len(); i++ {
			result := resultsValue.Index(i)
			_ = scope.Err(field.Set(result))
		}
	}
}

// handleHasManyPreload used to preload has many associations
func (scope *Scope) handleHasManyPreload(field *Field, conditions *Conditions) {
	relation := field.Relationship

	// get relations's primary keys
	primaryKeys := scope.getColumnAsArray(relation.AssociationForeignFieldNames, scope.Value)
	if len(primaryKeys) == 0 {
		return
	}

	// preload conditions
	preloadDB := conditions.MergeTo(scope.NewDB())

	// find relations
	query := fmt.Sprintf("%v IN (%v)", toQueryCondition(scope.db.dialect, relation.ForeignDBNames), toQueryMarks(primaryKeys))
	values := toQueryValues(primaryKeys)
	if relation.PolymorphicType != "" {
		query += fmt.Sprintf(" AND %v = ?", scope.Quote(relation.PolymorphicDBName))
		values = append(values, relation.PolymorphicValue)
	}

	results := makeSlice(field.Struct.Type)
	scope.Err(preloadDB.Where(query, values...).Find(results).Error)

	// assign find results
	var (
		resultsValue       = indirect(reflect.ValueOf(results))
		indirectScopeValue = scope.IndirectValue()
	)

	if indirectScopeValue.Kind() == reflect.Slice {
		preloadMap := make(map[string][]reflect.Value)
		for i := 0; i < resultsValue.Len(); i++ {
			result := resultsValue.Index(i)
			foreignValues := getValueFromFields(result, relation.ForeignFieldNames)
			preloadMap[toString(foreignValues)] = append(preloadMap[toString(foreignValues)], result)
		}

		for j := 0; j < indirectScopeValue.Len(); j++ {
			object := indirect(indirectScopeValue.Index(j))
			objectRealValue := getValueFromFields(object, relation.AssociationForeignFieldNames)
			f := object.FieldByName(field.Name)
			if results, ok := preloadMap[toString(objectRealValue)]; ok {
				f.Set(reflect.Append(f, results...))
			} else {
				f.Set(reflect.MakeSlice(f.Type(), 0, 0))
			}
		}
	} else {
		_ = scope.Err(field.Set(resultsValue))
	}
}

// handleBelongsToPreload used to preload belongs to associations
func (scope *Scope) handleBelongsToPreload(field *Field, conditions *Conditions) {
	relation := field.Relationship

	// preload conditions
	preloadDB := conditions.MergeTo(scope.NewDB())

	// get relations's primary keys
	primaryKeys := scope.getColumnAsArray(relation.ForeignFieldNames, scope.Value)
	if len(primaryKeys) == 0 {
		return
	}

	// find relations
	results := makeSlice(field.Struct.Type)
	_ = scope.Err(preloadDB.Where(fmt.Sprintf("%v IN (%v)",
		toQueryCondition(scope.db.dialect, relation.AssociationForeignDBNames),
		toQueryMarks(primaryKeys)),
		toQueryValues(primaryKeys)...).
		Find(results).Error)

	// assign find results
	var (
		resultsValue       = indirect(reflect.ValueOf(results))
		indirectScopeValue = scope.IndirectValue()
	)

	for i := 0; i < resultsValue.Len(); i++ {
		result := resultsValue.Index(i)
		if indirectScopeValue.Kind() == reflect.Slice {
			value := getValueFromFields(result, relation.AssociationForeignFieldNames)
			for j := 0; j < indirectScopeValue.Len(); j++ {
				object := indirect(indirectScopeValue.Index(j))
				if equalAsString(getValueFromFields(object, relation.ForeignFieldNames), value) {
					object.FieldByName(field.Name).Set(result)
				}
			}
		} else {
			scope.Err(field.Set(result))
		}
	}
}

// handleManyToManyPreload used to preload many to many associations
func (scope *Scope) handleManyToManyPreload(field *Field, conditions *Conditions) {
	var (
		relation         = field.Relationship
		joinTableHandler = relation.JoinTableHandler
		fieldType        = field.Struct.Type.Elem()
		linkHash         = map[string][]reflect.Value{}
		isPtr            bool
	)

	if fieldType.Kind() == reflect.Ptr {
		isPtr = true
		fieldType = fieldType.Elem()
	}

	var sourceKeys = joinTableHandler.SourceForeignKeys()

	// preload conditions
	preloadDB := conditions.MergeTo(scope.NewDB())

	// generate query with join table
	newScope := scope.New(reflect.New(fieldType).Interface())
	preloadDB = preloadDB.Table(newScope.TableName()).Model(newScope.Value)

	if len(preloadDB.search.selects) == 0 {
		preloadDB = preloadDB.Select(IQ("{}.*"))
	}

	preloadDB = joinTableHandler.JoinWith(joinTableHandler, preloadDB, scope.Value)
	rows, err := preloadDB.Rows()

	if scope.Err(err) != nil {
		return
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	for rows.Next() {
		var (
			elem     = reflect.New(fieldType).Elem()
			instance = InstanceOf(elem.Addr().Interface())
		)

		// register foreign keys in join tables
		var joinTableFields []*Field
		for _, key := range sourceKeys {
			joinTableFields = append(joinTableFields, &Field{
				StructField: &StructField{
					DBName:   key.DBName,
					IsNormal: true,
					Struct: reflect.StructField{
						Type: key.AssociationField.Struct.Type,
					},
					Assigner: key.AssociationField.Assigner,
				},
				Field: instance.FieldsMap[key.AssociationField.Name].Field,
			})
		}

		scope.scan(rows, columns, append(instance.Fields, joinTableFields...), nil)

		scope.New(elem.Addr().Interface()).
			InstanceSet("aorm:skip_query_callback", true).
			callCallbacks(scope.db.parent.callbacks.queries)

		var foreignKeys = make([]interface{}, len(sourceKeys))
		// generate hashed forkey keys in join table
		for idx, joinTableField := range joinTableFields {
			f := joinTableField.Field
			if f.Kind() == reflect.Ptr {
				if f.IsNil() {
					continue
				}
				f = f.Elem()
			}
			foreignKeys[idx] = f.Interface()
		}
		hashedSourceKeys := toString(foreignKeys)

		if isPtr {
			linkHash[hashedSourceKeys] = append(linkHash[hashedSourceKeys], elem.Addr())
		} else {
			linkHash[hashedSourceKeys] = append(linkHash[hashedSourceKeys], elem)
		}
	}

	if err := rows.Err(); err != nil {
		scope.Err(err)
	}

	// assign find results
	var (
		indirectScopeValue = scope.IndirectValue()
		fieldsSourceMap    = map[string][]reflect.Value{}
		foreignFieldNames  = []string{}
	)

	for _, fieldName := range relation.ForeignFieldNames {
		if _, ok := scope.instance.FieldsMap[fieldName]; ok {
			foreignFieldNames = append(foreignFieldNames, fieldName)
		}
	}

	if indirectScopeValue.Kind() == reflect.Slice {
		for j := 0; j < indirectScopeValue.Len(); j++ {
			object := indirect(indirectScopeValue.Index(j))
			key := toStringFields(object, foreignFieldNames)
			fieldsSourceMap[key] = append(fieldsSourceMap[key], object.FieldByName(field.Name))
		}
	} else if indirectScopeValue.IsValid() {
		key := toStringFields(indirectScopeValue, foreignFieldNames)
		fieldsSourceMap[key] = append(fieldsSourceMap[key], indirectScopeValue.FieldByName(field.Name))
	}
	for source, fields := range fieldsSourceMap {
		for _, f := range fields {
			//If not 0 this means Value is a pointer and we already added preloaded models to it
			if f.Len() != 0 {
				continue
			}

			v := reflect.MakeSlice(f.Type(), 0, 0)
			if len(linkHash[source]) > 0 {
				v = reflect.Append(f, linkHash[source]...)
			}

			f.Set(v)
		}
	}
}
