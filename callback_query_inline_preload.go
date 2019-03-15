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
	if _, skip := scope.InstanceGet("gorm:skip_query_callback"); skip {
		return
	}

	if scope.HasError() {
		return
	}

	scope.AutoInlinePreload()

	if len(scope.Search.inlinePreload) == 0 {
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
			var currentOptions *InlinePreloadOptions

			// if not preloaded
			if preloadKey := strings.Join(preloadFields[:idx+1], "."); !preloadedMap[preloadKey] {
				// assign search conditions to last preload
				if idx == len(preloadFields)-1 {
					currentOptions = preload.options
				}

				if field, ok := currentModelStruct.StructFieldsByName[preloadField]; ok && field.Relationship != nil && field.Relationship.Kind == "belongs_to" {
					currentIndex = append(currentIndex, field.StructIndex)
					currentScope = currentScope.getColumnAsScope(field.Name)
					currentScope.handleBelongsToInlinePreload(rootScope, scope, []string{}, field, currentIndex, currentOptions)
					currentModelStruct = currentScope.GetModelStruct()
					preloadedMap[preloadKey] = true
				} else if currentModelStruct.virtualFields[preloadField] != nil && currentModelStruct.virtualFields[preloadField] != nil {
					vf := currentModelStruct.virtualFields[preloadField]
					currentIndex = append(currentIndex, []int{(vf.StructIndex + 1) * -1})
					value := reflect.New(vf.ModelStruct.ModelType).Interface()
					currentScope = currentScope.New(value)
					currentScope.virtualFieldInlinePreload(rootScope, scope, []string{}, vf, currentIndex, currentOptions)
					currentModelStruct = vf.ModelStruct
					preloadedMap[preloadKey] = true
				} else {
					scope.Err(fmt.Errorf("can't inline preload field %s for %s", preloadField, currentModelStruct.ModelType))
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

	for i, fk := range relation.ForeignDBNames {
		query[i] = fmt.Sprintf("%v.%v = %v.%v", dbName, relation.AssociationForeignDBNames[i], rqtn, fk)
	}

	joinQuery := fmt.Sprintf("%s JOIN %v AS %v ON ", options.Join, qtn, dbName) + strings.Join(query, ", ")

	rootScope.Search.Joins(joinQuery)
	inlineRelated := &InlinePreloader{
		ID:        dbName,
		scope:     scope,
		rootScope: rootScope,
		Field:     field,
		Index:     index,
	}

	if options.Select != nil {
		inlineRelated.Fields(options.Select)
	}

	inlineRelated.Apply()

	for _, rf := range inlineRelated.RelationFields {
		scope.getColumnAsScope(rf.Name).handleBelongsToInlinePreload(rootScope, scope, path, rf, append(index, rf.StructIndex), &InlinePreloadOptions{})
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

	scope.Fields()
	scope.Search.Table(dbName)

	var fieldDbName string
	if field.LocalFieldName == "" {
		fieldDbName = parentScope.PrimaryField().DBName
	} else {
		fieldDbName = parentScope.MustFieldByName(field.LocalFieldName).DBName
	}

	builder.Prepare(func(c *Conditions, query interface{}, args []interface{}, replace func(query interface{}, args ...interface{})) {
		var replaced bool
		if f, ok := query.(func(info *InlinePreloadInfo, replace func(query interface{}, args ...interface{}))); ok {
			builder.InlinePreloadInfo.Conditions = c
			f(builder.InlinePreloadInfo, func(query interface{}, args ...interface{}) {
				replace(query, args...)
				replaced = true
			})
		} else if f, ok := query.(func(info *InlinePreloadInfo)); ok {
			builder.InlinePreloadInfo.Conditions = c
			f(builder.InlinePreloadInfo)
		}
		if !replaced {
			replace(nil)
		}
		builder.InlinePreloadInfo.Conditions = nil
	})

	builder.Where(fmt.Sprintf("%v.%v = %v.%v", rqtn, fieldDbName, dbName, scope.PrimaryField().DBName))

	where := builder.whereSQL(scope)
	joinQuery := fmt.Sprintf("%s JOIN %v AS %v ON ", options.Join, qtn, dbName) + where

	rootScope.Search.Joins(joinQuery, builder.Args...)

	inlineRelated := &InlinePreloader{
		ID:           dbName,
		scope:        scope,
		rootScope:    rootScope,
		VirtualField: field,
		Index:        index,
	}

	if options.Select != nil {
		inlineRelated.Fields(options.Select...)
	}

	inlineRelated.Apply()

	for _, rf := range inlineRelated.RelationFields {
		scope.getColumnAsScope(rf.Name).handleBelongsToInlinePreload(rootScope, scope, path, rf, append(index, rf.StructIndex), &InlinePreloadOptions{})
	}
}
