package aorm

import (
	"fmt"
	"reflect"
	"strings"
)

func InlinePreloadCallback(scope *Scope) {
	inlinePreloadCallback(scope)
}

// inlinePreloadCallback used to preload associations
func inlinePreloadCallback(scope *Scope) {
	if scope.HasError() || scope.Search.raw {
		return
	} else if _, skip := scope.InstanceGet(OptKeySkipPreload); skip {
		return
	}

	if scope.inlinePreloads == nil {
		scope.inlinePreloads = &InlinePreloads{DBNames: map[string]string{}}
	}

	scope.AutoInlinePreload()

	if len(scope.Search.inlinePreload) == 0 {
		return
	}

	currentScope := scope
	reflectedValue := reflect.Indirect(reflect.ValueOf(scope.Value))

	if IsManyResult(scope.Value) {
		reflectedValue = reflect.New(scope.Struct().Type)
		currentScope = scope.New(reflectedValue.Interface())
		currentScope.Search = scope.Search
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
			currentModelStruct = scope.Struct()
			currentIndex       = index[:]
		)

		for idx, preloadField := range preloadFields {
			var currentOptions *InlinePreloadOptions

			// if not preloaded
			if preloadKey := strings.Join(preloadFields[:idx+1], "."); !preloadedMap[preloadKey] {
				// assign search conditions to last preload
				if idx == len(preloadFields)-1 {
					currentOptions = preload.options
				}

				if field, ok := currentModelStruct.FieldsByName[preloadField]; ok && field.Relationship != nil && (field.Relationship.Kind == "belongs_to" || field.Relationship.Kind == "has_one") {
					currentIndex = append(currentIndex, field.StructIndex)
					currentScope = currentScope.ScopeOfField(field.Name)
					currentScope.handleBelongsToInlinePreload(rootScope, scope, []string{}, field, currentIndex, currentOptions)
					currentModelStruct = currentScope.Struct()
					preloadedMap[preloadKey] = true
				} else if currentModelStruct.virtualFields[preloadField] != nil {
					vf := currentModelStruct.virtualFields[preloadField]
					currentIndex = append(currentIndex, []int{(vf.StructIndex + 1) * -1})
					value := reflect.New(vf.Model.Type).Interface()
					currentScope = currentScope.New(value)
					currentScope.virtualFieldInlinePreload(rootScope, scope, []string{}, vf, currentIndex, currentOptions)
					currentModelStruct = vf.Model
					preloadedMap[preloadKey] = true
				} else {
					scope.Err(fmt.Errorf("can't inline preload field %s for %s", preloadField, currentModelStruct.Type))
					return
				}
			}
		}
	}
}

// handleBelongsToPreload used to preload belongs to associations
func (scope *Scope) handleBelongsToInlinePreload(rootScope, parentScope *Scope, path []string, field *StructField, index [][]int, options *InlinePreloadOptions) {
	path = append(path, field.Name)
	var (
		relation = field.Relationship
		query    = make([]string, len(relation.ForeignFieldNames))
		rqtn     = parentScope.QuotedTableName()
		qtn      = scope.QuotedTableName()
		dbName   = rootScope.inlinePreloads.Next(path...)
	)
	scope.Search.Table(dbName)

	if field.IsChild {
		scope.Search.ignorePrimaryFields = true
	}

	for i, fk := range relation.ForeignDBNames {
		query[i] = fmt.Sprintf("%v.%v = %v.%v", dbName, relation.AssociationForeignDBNames[i], rqtn, fk)
	}

	joinQuery := fmt.Sprintf("%s JOIN %v AS %v ON ", options.Join, qtn, dbName) + strings.Join(query, ", ")

	rootScope.Search.Joins(joinQuery)
	inlineRelated := &InlinePreloader{
		ID:          dbName,
		Scope:       scope,
		RootScope:   rootScope,
		ParentScope: parentScope,
		Field:       field,
		Index:       index,
	}

	if !rootScope.counter {
		if len(options.Select) > 0 {
			inlineRelated.Fields(options.Select...)
		}

		inlineRelated.Select()
	}

	for _, rf := range inlineRelated.RelationFields {
		scope.ScopeOfField(rf.Name).handleBelongsToInlinePreload(rootScope, scope, path, rf, append(index, rf.StructIndex), &InlinePreloadOptions{})
	}
}

// handleBelongsToPreload used to preload belongs to associations
func (scope *Scope) virtualFieldInlinePreload(rootScope, parentScope *Scope, path []string, field *VirtualField, index [][]int, options *InlinePreloadOptions) {
	path = append(path, field.FieldName)
	var (
		rqtn    = parentScope.QuotedTableName()
		qtn     = scope.QuotedTableName()
		dbName  = rootScope.inlinePreloads.Next(path...)
		builder = &InlinePreloadBuilder{&options.Conditions, &InlinePreloadInfo{RootScope: rootScope, ParentScope: parentScope, Scope: scope}}
	)

	scope.Instance()
	scope.Search.Table(dbName)

	if parentScope.HasPrimaryFields() {
		var fieldDbName string
		if field.LocalFieldName == "" {
			fieldDbName = parentScope.PrimaryField().DBName
		} else {
			fieldDbName = parentScope.MustFieldByName(field.LocalFieldName).DBName
		}

		builder.Prepare(func(c *Conditions, query interface{}, args []interface{}, replace func(query interface{}, args ...interface{})) {
			if f, ok := query.(func(info *InlinePreloadInfo, replace func(query interface{}, args ...interface{}))); ok {
				builder.InlinePreloadInfo.Conditions = c
				f(builder.InlinePreloadInfo, func(query interface{}, args ...interface{}) {
					replace(query, args...)
				})
			} else if f, ok := query.(func(info *InlinePreloadInfo)); ok {
				builder.InlinePreloadInfo.Conditions = c
				f(builder.InlinePreloadInfo)
			}
			builder.InlinePreloadInfo.Conditions = nil
		})

		builder.Where(fmt.Sprintf("%v.%v = %v.%v", rqtn, fieldDbName, dbName, scope.PrimaryField().DBName))

		where := builder.WhereClause(scope)
		where.Query = fmt.Sprintf("%s JOIN %v AS %v ON ", options.Join, qtn, dbName) + where.Query
		rootScope.Search.Joins(where.Query, where.Args...)
	} else {
		joinQuery := fmt.Sprintf("%s JOIN %v AS %v ON 1 = 1", options.Join, qtn, dbName)
		rootScope.Search.Joins(joinQuery, builder.Args...)
	}

	inlineRelated := &InlinePreloader{
		ID:           dbName,
		ParentScope:  parentScope,
		Scope:        scope,
		RootScope:    rootScope,
		VirtualField: field,
		Index:        index,
	}

	if !rootScope.counter {
		if len(options.Select) > 0 {
			inlineRelated.Fields(options.Select...)
		}

		inlineRelated.Select()
	}

	for _, rf := range inlineRelated.RelationFields {
		scope.ScopeOfField(rf.Name).handleBelongsToInlinePreload(rootScope, scope, path, rf, append(index, rf.StructIndex), &InlinePreloadOptions{})
	}
}
