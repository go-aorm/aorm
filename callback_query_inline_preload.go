package gorm

import (
	"fmt"
	"reflect"
	"strings"
)

// inlinePreloadCallback used to preload associations
func inlinePreloadCallback(scope *Scope) {
	if _, skip := scope.InstanceGet("gorm:skip_query_callback"); skip {
		return
	}

	if scope.Search.inlinePreload == nil || scope.HasError() {
		return
	}

	var counter *InlinePreloadCounter
	if counterInterface, ok := scope.db.Get("gorm:inline_related:counter"); ok {
		counter = counterInterface.(*InlinePreloadCounter)
	} else {
		counter = &InlinePreloadCounter{}
		scope.db = scope.db.Set("gorm:inline_related:counter", counter)
	}

	currentScope := scope
	reflectedValue := reflect.Indirect(reflect.ValueOf(scope.Value))

	if reflectedValue.Kind() == reflect.Slice {
		reflectedValue = reflect.New(scope.GetModelStruct().ModelType)
		currentScope = scope.New(reflectedValue.Interface())
		currentScope.Search = scope.Search
	}

	inlinePreload(scope, currentScope, counter, [][]int{})
}

// inlinePreloadCallback used to preload associations
func inlinePreload(rootScope, scope *Scope, counter *InlinePreloadCounter, index [][]int) {
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
					currentScope.handleBelongsToInlinePreload(rootScope, scope, counter, field, currentIndex, currentPreloadConditions)
					currentModelStruct = currentScope.GetModelStruct()
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
func (scope *Scope) handleBelongsToInlinePreload(rootScope, parentScope *Scope, counter *InlinePreloadCounter, field *StructField, index [][]int, conditions []interface{}) {
	var (
		relation = field.Relationship
		query    = make([]string, len(relation.ForeignFieldNames))
		rqtn     = parentScope.QuotedTableName()
		qtn      = scope.QuotedTableName()
		id       = "gorm_prl_" + counter.NextS()
	)

	scope.Search.Table(id)

	for i, fk := range relation.ForeignDBNames {
		query[i] = fmt.Sprintf("%v.%v = %v.%v", id, relation.AssociationForeignDBNames[i], rqtn, fk)
	}

	joinQuery := fmt.Sprintf("LEFT JOIN %v AS %v ON ", qtn, id) + strings.Join(query, ", ")
	rootScope.Search.Joins(joinQuery)
	inlineRelated := &InlinePreloader{
		ID:        id,
		scope:     scope,
		rootScope: rootScope,
		Field:     field,
		Index:     index,
	}

	inlineRelated.Apply()

	for _, rf := range inlineRelated.RelationFields {
		scope.getColumnAsScope(rf.Name).handleBelongsToInlinePreload(rootScope, scope, counter, rf, append(index, rf.StructIndex), []interface{}{})
	}
}
