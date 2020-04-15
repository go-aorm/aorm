package aorm

import (
	"reflect"

	"github.com/pkg/errors"
)

var modelStructStorage = NewModelStructStorage()

type ModelStructStorage struct {
	ModelStructsMap    *safeModelStructsMap
	TableNameResolvers []TableNameResolver
	GetAssigner        func(typ reflect.Type) Assigner
}

func NewModelStructStorage() *ModelStructStorage {
	return &ModelStructStorage{ModelStructsMap: modelStructsMap, GetAssigner: assigners.Get}
}

func (this *ModelStructStorage) GetOrNew(value interface{}, callback ...func(modelStruct *ModelStruct)) (modelStruct *ModelStruct, err error) {
	return this.getOrNew(value, false, callback...)
}

func (this *ModelStructStorage) getOrNew(value interface{}, embedded bool, callback ...func(modelStruct *ModelStruct)) (modelStruct *ModelStruct, err error) {
	modelStruct = &ModelStruct{}
	// value can'T be nil
	if value == nil {
		return
	}

	var reflectType reflect.Type
	if reflectType = AcceptableTypeForModelStructInterface(value); reflectType == nil {
		return nil, errors.New("bad value type")
	}

	// Get Cached model struct
	if value := this.ModelStructsMap.Get(reflectType); value != nil {
		return value, nil
	}

	modelStruct.Value = reflect.New(reflectType).Interface()
	modelStruct.Type = reflectType
	modelStruct.storage = this
	modelStruct.Indexes = make(IndexMap)
	modelStruct.UniqueIndexes = make(IndexMap)

	this.ModelStructsMap.Set(reflectType, modelStruct)

	if err = modelStruct.setup(embedded); err != nil {
		return nil, err
	}
	return
}
