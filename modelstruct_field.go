package aorm

import (
	"fmt"
	"reflect"
	"strconv"
)

type StructFieldMethodCallback struct {
	Method
	Caller reflect.Value
}

func (s StructFieldMethodCallback) Call(object reflect.Value, in []reflect.Value) {
	s.Caller.Call(append([]reflect.Value{reflect.ValueOf(&s.Method), s.ObjectMethod(object)}, in...))
}

// StructField model field's struct definition
type StructField struct {
	DBName          string
	Name            string
	Names           []string
	IsPrimaryKey    bool
	IsChild         bool
	IsNormal        bool
	IsIgnored       bool
	IsScanner       bool
	IsReadOnly      bool
	HasDefaultValue bool
	Tag             reflect.StructTag
	TagSettings     TagSetting
	Struct          reflect.StructField
	BaseModel       *ModelStruct
	Model           *ModelStruct
	IsForeignKey    bool
	Relationship    *Relationship
	MethodCallbacks map[string]StructFieldMethodCallback
	StructIndex     []int
	Index           int
	Assigner        Assigner
	Data            map[interface{}]interface{}
	Selector        FieldSelector
	SelectWraper    FieldSelectWraper
	Flag            FieldFlag
}

func (this *StructField) String() string {
	return this.BaseModel.Fqn() + "#" + this.Name
}

func (this *StructField) Structure() *FieldStructure {
	return &FieldStructure{this.Struct.Type, this.TagSettings, this.Assigner, this.IsPrimaryKey}
}

// Call the method callback if exists by name.
func (this *StructField) CallMethodCallbackArgs(name string, object reflect.Value, in []reflect.Value) {
	if callback, ok := this.MethodCallbacks[name]; ok {
		callback.Call(object, in)
	}
}

// Call the method callback if exists by name. the
func (this *StructField) CallMethodCallback(name string, object reflect.Value, in ...reflect.Value) {
	this.CallMethodCallbackArgs(name, object, in)
}

func (this StructField) clone() *StructField {
	clone := this
	clone.TagSettings = map[string]string{}
	clone.Data = map[interface{}]interface{}{}

	if this.Relationship != nil {
		relationship := *this.Relationship
		clone.Relationship = &relationship
	}

	for key, value := range this.TagSettings {
		clone.TagSettings[key] = value
	}

	for key, value := range this.Data {
		clone.Data[key] = value
	}

	return &clone
}

func (this StructField) IDOf(v interface{}) (IDValuer, error) {
	var value reflect.Value
	if v2, ok := v.(reflect.Value); ok {
		value = v2
	} else {
		value = reflect.ValueOf(v)
	}
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil, nil
		}
		value = value.Elem()
	}
	typ := indirectType(value.Type())
	if typ != this.Struct.Type && typ.AssignableTo(this.Struct.Type) {
		if typ.ConvertibleTo(this.Struct.Type) {
			value = value.Convert(this.Struct.Type)
		} else {
			return nil, fmt.Errorf("%s#%s. can't converts from %T to %s", this.BaseModel.Fqn(), this.Struct.Name, v, this.Struct.Type)
		}
	}
	if typ.Implements(reflect.TypeOf((*IDValuer)(nil)).Elem()) {
		return value.Interface().(IDValuer), nil
	}
	var i = value.Interface()
	switch t := i.(type) {
	case int, int8, int16, int32, int64:
		return IntId(value.Int()), nil
	case uint, uint8, uint16, uint32, uint64:
		return UintId(value.Uint()), nil
	case string:
		return StrId(t), nil
	case []byte:
		return BytesId(value.Interface().([]byte)), nil
	default:
		if typ.Implements(reflect.TypeOf((*IDValueRawConverter)(nil)).Elem()) {
			return reflect.New(typ).Interface().(IDValueRawConverter).FromRaw(i), nil
		}
		if this.Assigner != nil {
			if conv, ok := this.Assigner.(IDValueRawConverter); ok {
				return conv.FromRaw(i), nil
			}
		}
		if conv := IDValueRawConverterGet(typ); conv != nil {
			return conv.FromRaw(i), nil
		}
		return nil, fmt.Errorf("%s#%s. can't converts from raw %T to aorm.IDValuer because "+
			"IDValueRawConverter not registered in Fielt.Type (assignable) or Field.Assigner "+
			"(assignable) or aorm.IDValueRawConverterRegister",
			this.BaseModel.Fqn(), this.Struct.Name, v)
	}
}

func (this StructField) DefaultID() (IDValuer, error) {
	return this.IDOf(reflect.New(this.Struct.Type).Interface())
}

func (this StructField) TextSize() (size int) {
	if num, ok := this.TagSettings["SIZE"]; ok {
		size, _ = strconv.Atoi(num)
	} else if sizer, ok := this.Assigner.(SQLSizer); ok {
		size = sizer.SQLSize(nil)
	} else {
		if ui16 := TextSize(this.TagSettings["TYPE"]); size == 0 {
			size = 255
		} else {
			size = int(ui16)
		}
	}
	return
}
