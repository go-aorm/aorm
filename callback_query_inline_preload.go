package aorm

import (
	"fmt"
	"reflect"
	"strings"
)

type InlinePreloadOption uint

func (ipo InlinePreloadOption) IsInnerJoin() bool {
	if (ipo & IPO_INNER_JOIN) != 0 {
		return true
	}
	return false
}

const (
	IPO_INNER_JOIN InlinePreloadOption = 1 << iota
)

func InlinePreloadCallback(scope *Scope) {
	inlinePreloadCallback(scope)
}

// inlinePreloadCallback used to preload associations
func inlinePreloadCallback(scope *Scope) {
	if _, skip := scope.InstanceGet("gorm:skip_query_callback"); skip {
		return
	}

	if scope.Search.inlinePreload == nil || scope.HasError() {
		return
	}

	currentScope := scope
	reflectedValue := reflect.Indirect(reflect.ValueOf(scope.Value))

	if reflectedValue.Kind() == reflect.Slice {
		reflectedValue = reflect.New(scope.GetModelStruct().ModelType)
		currentScope = scope.New(reflectedValue.Interface())
		currentScope.Search = scope.Search
	}

	if scope.inlinePreloads == nil {
		scope.inlinePreloads = &InlinePreloads{DBNames: map[string]string{}}
	}

	scope.inlinePreloads.DBNames["{}"] = scope.TableName()

	inlinePreload(scope, currentScope, [][]int{})
}

// inlinePreloadCallback used to preload associations
func inlinePreload(rootScope, scope *Scope, index [][]int) {
	preloadedMap := map[string]bool{}

	for _, preload := range scope.Search.inlinePreload {
		var (
			preloadFields      = strings.Split(preload.schema, ".")
			currentScope       = scope
			currentModelStruct = scope.GetModelStruct()
			currentIndex       = index[:]
		)

		for idx, preloadField := range preloadFields {
			var currentPreloadConditions []interface{}

			// if not preloaded
			if preloadKey := strings.Join(preloadFields[:idx+1], "."); !preloadedMap[preloadKey] {
				// assign search conditions to last preload
				if idx == len(preloadFields)-1 {
					currentPreloadConditions = preload.conditions
				}

				if field, ok := currentModelStruct.StructFieldsByName[preloadField]; ok && field.Relationship != nil && field.Relationship.Kind == "belongs_to" {
					currentIndex = append(currentIndex, field.StructIndex)
					currentScope = currentScope.getColumnAsScope(field.Name)
					currentScope.handleBelongsToInlinePreload(rootScope, scope, []string{}, field, currentIndex, currentPreloadConditions)
					currentModelStruct = currentScope.GetModelStruct()
					preloadedMap[preloadKey] = true
				} else if currentModelStruct.virtualFields[preloadField] != nil && currentModelStruct.virtualFields[preloadField] != nil {
					vf := currentModelStruct.virtualFields[preloadField]
					currentIndex = append(currentIndex, []int{(vf.StructIndex + 1) * -1})
					value := reflect.New(vf.ModelStruct.ModelType).Interface()
					currentScope = currentScope.New(value)
					currentScope.virtualFieldInlinePreload(rootScope, scope, []string{}, vf, currentIndex, currentPreloadConditions)
					currentModelStruct = vf.ModelStruct
					preloadedMap[preloadKey] = true
				} else {
					scope.Err(fmt.Errorf("can't preload field %s for %s", preloadField, currentModelStruct.ModelType))
					return
				}
			}
		}
	}
}

// handleBelongsToPreload used to preload belongs to associations
func (scope *Scope) handleBelongsToInlinePreload(rootScope, parentScope *Scope, path []string, field *StructField, index [][]int, conditions []interface{}) {
	path = append(path, field.Name)
	var (
		relation  = field.Relationship
		query     = make([]string, len(relation.ForeignFieldNames))
		rqtn      = parentScope.QuotedTableName()
		qtn       = scope.QuotedTableName()
		dbName    = rootScope.inlinePreloads.Next(path...)
		fields    []string
		innerJoin bool
	)

	if len(conditions) > 0 {
		if ipo, ok := conditions[0].(InlinePreloadOption); ok {
			innerJoin = ipo.IsInnerJoin()
			conditions = conditions[1:]
		}

		for _, c := range conditions {
			switch ct := c.(type) {
			case []string:
				fields = ct
			}
		}
	}

	scope.Search.Table(dbName)

	for i, fk := range relation.ForeignDBNames {
		query[i] = fmt.Sprintf("%v.%v = %v.%v", dbName, relation.AssociationForeignDBNames[i], rqtn, fk)
	}

	joinQuery := fmt.Sprintf("JOIN %v AS %v ON ", qtn, dbName) + strings.Join(query, ", ")
	if !innerJoin {
		joinQuery = "LEFT " + joinQuery
	}
	rootScope.Search.Joins(joinQuery)
	inlineRelated := &InlinePreloader{
		ID:        dbName,
		scope:     scope,
		rootScope: rootScope,
		Field:     field,
		Index:     index,
	}

	if fields != nil {
		inlineRelated.Fields(fields)
	}

	inlineRelated.Apply()

	for _, rf := range inlineRelated.RelationFields {
		scope.getColumnAsScope(rf.Name).handleBelongsToInlinePreload(rootScope, scope, path, rf, append(index, rf.StructIndex), []interface{}{})
	}
}

// handleBelongsToPreload used to preload belongs to associations
func (scope *Scope) virtualFieldInlinePreload(rootScope, parentScope *Scope, path []string, field *VirtualField, index [][]int, conditions []interface{}) {
	path = append(path, field.FieldName)
	var (
		query     string
		rqtn      = parentScope.QuotedTableName()
		qtn       = scope.QuotedTableName()
		dbName    = rootScope.inlinePreloads.Next(path...)
		fields    []string
		innerJoin bool
	)

	if len(conditions) > 0 {
		if ipo, ok := conditions[0].(InlinePreloadOption); ok {
			innerJoin = ipo.IsInnerJoin()
			conditions = conditions[1:]
		}

		for _, c := range conditions {
			switch ct := c.(type) {
			case []string:
				fields = ct
			}
		}
	}

	scope.Fields()
	scope.Search.Table(dbName)
	query = fmt.Sprintf("%v.%v = %v.%v", dbName, scope.PrimaryField().DBName, rqtn, parentScope.PrimaryField().DBName)

	joinQuery := fmt.Sprintf("JOIN %v AS %v ON ", qtn, dbName) + query

	if !innerJoin {
		joinQuery = "LEFT " + joinQuery
	}

	rootScope.Search.Joins(joinQuery)
	inlineRelated := &InlinePreloader{
		ID:           dbName,
		scope:        scope,
		rootScope:    rootScope,
		VirtualField: field,
		Index:        index,
	}

	if fields != nil {
		inlineRelated.Fields(fields)
	}

	inlineRelated.Apply()

	for _, rf := range inlineRelated.RelationFields {
		scope.getColumnAsScope(rf.Name).handleBelongsToInlinePreload(rootScope, scope, path, rf, append(index, rf.StructIndex), []interface{}{})
	}
}
