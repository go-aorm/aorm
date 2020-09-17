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

func (this *ModelStructStorage) GetOrNew(value interface{}) (modelStruct *ModelStruct, err error) {
	return this.getOrNew(value, nil, false, nil)
}

func (this *ModelStructStorage) getOrNew(value interface{}, pth []string, embedded bool, from *ModelStruct) (modelStruct *ModelStruct, err error) {
	// value can'T be nil
	if value == nil {
		return
	}

	var reflectType reflect.Type
	if reflectType = AcceptableTypeForModelStructInterface(value); reflectType == nil {
		t := indirectType(reflect.TypeOf(value))
		return nil, errors.New("bad value type: " + t.PkgPath() + "." + t.Name())
	}
	reflectType = indirectType(reflectType)

	// Get Cached model struct
	if value := this.ModelStructsMap.Get(reflectType); value != nil {
		return value, nil
	}

	if modelStruct, err = this.create(reflectType); err != nil {
		return
	}

	modelStruct.parentTemp = from
	if err = modelStruct.setup(pth, embedded, from); err != nil {
		return nil, err
	}
	modelStruct.parentTemp = nil
	return
}

func (this *ModelStructStorage) create(value interface{}) (modelStruct *ModelStruct, err error) {
	var (
		reflectType reflect.Type
		ok          bool
	)
	if reflectType, ok = value.(reflect.Type); !ok {
		if reflectType = AcceptableTypeForModelStructInterface(value); reflectType == nil {
			return nil, errors.New("bad value type")
		}
	} else {
		reflectType = indirectType(reflectType)
	}
	modelStruct = new(ModelStruct)
	modelStruct.Value = reflect.New(reflectType).Interface()
	modelStruct.Type = reflectType
	modelStruct.storage = this
	modelStruct.Indexes = make(IndexMap)
	modelStruct.UniqueIndexes = make(IndexMap)
	modelStruct.ChildrenByName = make(map[string]*ModelStruct)
	modelStruct.HasManyChildrenByName = make(map[string]*ModelStruct)
	modelStruct.FieldsByName = make(map[string]*StructField)
	modelStruct.DynamicFieldsByName = make(map[string]*StructField)
	modelStruct.Tags = make(TagSetting)

	if f, ok := reflectType.FieldByName("_"); ok && len(f.Index) == 1 {
		modelStruct.Tags = parseFieldTagSetting(f)
	}

	if !modelStruct.Tags.Flag("INLINE") {
		this.ModelStructsMap.Set(reflectType, modelStruct)
		modelStruct.Name = reflectType.Name()
	}
	return
}
