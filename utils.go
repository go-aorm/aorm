package aorm

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// NowFunc returns current time, this function is exported in order to be able
// to give the flexibility to the developer to customize it according to their
// needs, e.g:
//    aorm.NowFunc = func() time.Time {
//      return time.Now().UTC()
//    }
var NowFunc = func() time.Time {
	return time.Now()
}

// Copied from golint
var commonInitialisms = []string{"API", "ASCII", "CPU", "CSS", "DNS", "EOF", "GUID", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "LHS", "QPS", "RAM", "RHS", "RPC", "SLA", "SMTP", "SSH", "TLS", "TTL", "UID", "UI", "UUID", "URI", "URL", "UTF8", "VM", "XML", "XSRF", "XSS"}
var commonInitialismsReplacer *strings.Replacer

var goSrcRegexp = regexp.MustCompile(`moisespsena-go/aorm(@.*)?/.*.go`)
var goTestRegexp = regexp.MustCompile(`moisespsena-go/aorm(@.*)?/.*test.go`)

func init() {
	var commonInitialismsForReplacer []string
	for _, initialism := range commonInitialisms {
		commonInitialismsForReplacer = append(commonInitialismsForReplacer, initialism, strings.Title(strings.ToLower(initialism)))
	}
	commonInitialismsReplacer = strings.NewReplacer(commonInitialismsForReplacer...)
}

type KVStorager interface {
	Set(key string, value string)
	Get(key string) string
}

type safeMap struct {
	m map[string]string
	l *sync.RWMutex
}

func (s *safeMap) Set(key string, value string) {
	s.l.Lock()
	defer s.l.Unlock()
	s.m[key] = value
}

func (s *safeMap) Get(key string) string {
	s.l.RLock()
	defer s.l.RUnlock()
	return s.m[key]
}

func newSafeMap() *safeMap {
	return &safeMap{l: new(sync.RWMutex), m: make(map[string]string)}
}

var smap = newSafeMap()

type strCase bool

const (
	lower strCase = false
	upper strCase = true
)

type SafeNameBuilder struct {
	Cache             KVStorager
	PreFormat, Format func(v string) string
}

func NewSafeNameBuilder(cache KVStorager, format ...func(v string) string) *SafeNameBuilder {
	if cache == nil {
		cache = smap
	}
	s := &SafeNameBuilder{Cache: cache}
	for _, s.Format = range format {
	}
	return s

}

func (this *SafeNameBuilder) Build(name string) string {
	if name == "" {
		return ""
	}
	if v := this.Cache.Get(name); v != "" {
		return v
	}
	s := this.build(name)
	this.Cache.Set(name, s)
	return s
}

func (this *SafeNameBuilder) BuildParts(name string, sep string) string {
	if name == "" {
		return ""
	}
	if v := this.Cache.Get(name); v != "" {
		return v
	}
	parts := strings.Split(name, sep)
	for i := range parts {
		parts[i] = this.build(parts[i])
	}
	s := strings.Join(parts, sep)
	this.Cache.Set(name, s)
	return s
}

func (this *SafeNameBuilder) build(name string) string {
	if this.PreFormat != nil {
		name = this.PreFormat(name)
	}
	var (
		value = commonInitialismsReplacer.Replace(name)
		buf   = bytes.NewBufferString("")

		lastCase, currCase,
		nextCase, nextNumber strCase
	)

	for i, v := range value[:len(value)-1] {
		nextCase = strCase(value[i+1] >= 'A' && value[i+1] <= 'Z')
		nextNumber = strCase(value[i+1] >= '0' && value[i+1] <= '9')

		if i > 0 {
			if currCase == upper {
				if lastCase == upper && (nextCase == upper || nextNumber == upper) {
					buf.WriteRune(v)
				} else {
					if value[i-1] != '_' && value[i+1] != '_' {
						buf.WriteRune('_')
					}
					buf.WriteRune(v)
				}
			} else {
				buf.WriteRune(v)
				if i == len(value)-2 && (nextCase == upper && nextNumber == lower) {
					buf.WriteRune('_')
				}
			}
		} else {
			currCase = upper
			buf.WriteRune(v)
		}
		lastCase = currCase
		currCase = nextCase
	}

	buf.WriteByte(value[len(value)-1])

	s := strings.ToLower(buf.String())
	if this.Format != nil {
		s = this.Format(s)
	}
	return s
}

var snb = NewSafeNameBuilder(nil)

// ToDBName convert string to db name
func ToDBName(name string) string {
	if v := smap.Get(name); v != "" {
		return v
	}

	if name == "" {
		return ""
	}

	var (
		value                                    = commonInitialismsReplacer.Replace(name)
		buf                                      = bytes.NewBufferString("")
		lastCase, currCase, nextCase, nextNumber strCase
	)

	for i, v := range value[:len(value)-1] {
		nextCase = strCase(value[i+1] >= 'A' && value[i+1] <= 'Z')
		nextNumber = strCase(value[i+1] >= '0' && value[i+1] <= '9')

		if i > 0 {
			if currCase == upper {
				if lastCase == upper && (nextCase == upper || nextNumber == upper) {
					buf.WriteRune(v)
				} else {
					if value[i-1] != '_' && value[i+1] != '_' {
						buf.WriteRune('_')
					}
					buf.WriteRune(v)
				}
			} else {
				buf.WriteRune(v)
				if i == len(value)-2 && (nextCase == upper && nextNumber == lower) {
					buf.WriteRune('_')
				}
			}
		} else {
			currCase = upper
			buf.WriteRune(v)
		}
		lastCase = currCase
		currCase = nextCase
	}

	buf.WriteByte(value[len(value)-1])

	s := strings.ToLower(buf.String())
	smap.Set(name, s)
	return s
}

// Expr generate raw SQL expression, for example:
//     DB.Model(&product).Update("price", aorm.Expr("price * ? + ?", 2, 100))
func Expr(expression string, args ...interface{}) *Query {
	return &Query{expression, args}
}

func indirect(reflectValue reflect.Value) reflect.Value {
	for reflectValue.Kind() == reflect.Ptr || reflectValue.Kind() == reflect.Interface {
		reflectValue = reflectValue.Elem()
	}
	return reflectValue
}

func indirectType(reflectType reflect.Type) reflect.Type {
	for reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}
	return reflectType
}

func indirectRealType(reflectType reflect.Type) reflect.Type {
	for reflectType.Kind() == reflect.Ptr || reflectType.Kind() == reflect.Interface {
		reflectType = reflectType.Elem()
	}
	return reflectType
}

func ptrToType(reflectType reflect.Type) reflect.Type {
	reflectType = indirectType(reflectType)
	return reflect.PtrTo(reflectType)
}

type Method struct {
	index int
	name  string
	ptr   bool
	valid bool
}

func (m Method) Index() int {
	return m.index
}

func (m Method) Name() string {
	return m.name
}

func (m Method) Ptr() bool {
	return m.ptr
}

func (m Method) Valid() bool {
	return m.valid
}

func (m Method) TypeMethod(typ reflect.Type) reflect.Method {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if m.ptr {
		typ = reflect.PtrTo(typ)
	}
	return typ.Method(m.index)
}

func (m Method) ObjectMethod(object reflect.Value) reflect.Value {
	object = indirect(object)
	if m.ptr {
		object = object.Addr()
	}
	return object.Method(m.index)
}

func MethodByName(typ reflect.Type, name string) (m Method) {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return
	}

	if method, ok := typ.MethodByName(name); ok {
		m.index = method.Index
		m.name = name
		m.valid = true
		return
	}

	if method, ok := reflect.PtrTo(typ).MethodByName(name); ok {
		m.index = method.Index
		m.name = name
		m.ptr = true
		m.valid = true
	}

	return
}

func toQueryMarks(primaryValues [][]interface{}) string {
	var results []string

	for _, primaryValue := range primaryValues {
		var marks []string
		for range primaryValue {
			marks = append(marks, "?")
		}

		if len(marks) > 1 {
			results = append(results, fmt.Sprintf("(%v)", strings.Join(marks, ",")))
		} else {
			results = append(results, strings.Join(marks, ""))
		}
	}
	return strings.Join(results, ",")
}

func toQueryCondition(scope *Scope, columns []string) string {
	var newColumns []string
	for _, column := range columns {
		newColumns = append(newColumns, scope.Quote(column))
	}

	if len(columns) > 1 {
		return fmt.Sprintf("(%v)", strings.Join(newColumns, ","))
	}
	return strings.Join(newColumns, ",")
}

func toQueryValues(values [][]interface{}) (results []interface{}) {
	for _, value := range values {
		for _, v := range value {
			results = append(results, v)
		}
	}
	return
}

func fileWithLineNum() string {
	for i := 2; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && (!goSrcRegexp.MatchString(file) || goTestRegexp.MatchString(file)) {
			return fmt.Sprintf("%v:%v", file, line)
		}
	}
	return ""
}

func IsBlank(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.String:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return value.IsNil()
	}

	return reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface())
}

func toSearchableMap(attrs ...interface{}) (result interface{}) {
	if len(attrs) > 1 {
		if str, ok := attrs[0].(string); ok {
			result = map[string]interface{}{str: attrs[1]}
		}
	} else if len(attrs) == 1 {
		if attr, ok := attrs[0].(map[string]interface{}); ok {
			result = attr
		}

		if attr, ok := attrs[0].(interface{}); ok {
			result = attr
		}
	}
	return
}

func equalAsString(a interface{}, b interface{}) bool {
	return toString(a) == toString(b)
}

func toString(str interface{}) string {
	switch t := str.(type) {
	case []interface{}:
		var results []string
		for _, value := range t {
			results = append(results, toString(value))
		}
		return strings.Join(results, "_")
	case []byte:
		return string(t)
	case interface{ AsBytes() []byte }:
		return toString(t.AsBytes())
	case interface{ Bytes() []byte }:
		return toString(t.Bytes())
	case reflect.Value:
		if t.IsValid() {
			switch t.Kind() {
			case reflect.Array:
				l := t.Len()
				var newV = reflect.New(t.Type())
				newV.Elem().Set(t)
				t = newV.Elem().Slice(0, l)
				return toString(t.Interface())
			default:
				if t = reflect.Indirect(t); t.IsValid() {
					return fmt.Sprintf("%v", t.Interface())
				}
			}
		}
	default:
		return toString(reflect.ValueOf(str))
	}
	return ""
}

// TupleQueryArgs create tuple args from argCount
func TupleQueryArgs(argCount int) (query string) {
	if argCount == 0 {
		return
	}
	query = strings.Repeat("?,", argCount)
	query = query[0 : len(query)-1]
	return "(" + query + ")"
}

func makeSlice(elemType reflect.Type) interface{} {
	return makeSliceValue(elemType).Interface()
}

func makeSliceValue(elemType reflect.Type) reflect.Value {
	if elemType.Kind() == reflect.Slice {
		elemType = elemType.Elem()
	}
	sliceType := reflect.SliceOf(elemType)
	slice := reflect.New(sliceType)
	slice.Elem().Set(reflect.MakeSlice(sliceType, 0, 0))
	return slice
}

func strInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// getValueFromFields return given fields's value
func getValueFromFields(value reflect.Value, fieldNames []string) (results []interface{}) {
	// If value is a nil pointer, Indirect returns a zero Value!
	// Therefor we need to check for a zero value,
	// as CreateFieldByName could panic
	if indirectValue := reflect.Indirect(value); indirectValue.IsValid() {
		for _, fieldName := range fieldNames {
			if fieldValue := indirectValue.FieldByName(fieldName); fieldValue.IsValid() {
				result := fieldValue.Interface()
				if r, ok := result.(driver.Valuer); ok {
					result, _ = r.Value()
				}
				results = append(results, result)
			}
		}
	}
	return
}

// toStringFields return given fields's as string value
func toStringFields(value reflect.Value, fieldNames []string) string {
	var w bytes.Buffer
	// If value is a nil pointer, Indirect returns a zero Value!
	// Therefor we need to check for a zero value,
	// as CreateFieldByName could panic
	if indirectValue := reflect.Indirect(value); indirectValue.IsValid() {
		for _, fieldName := range fieldNames {
			if fieldValue := indirectValue.FieldByName(fieldName); fieldValue.IsValid() {
				s := toString(fieldValue.Interface())
				if s == "" {
					continue
				}
				if w.Len() == 0 {
					w.WriteString(s)
				} else {
					w.WriteString("_" + s)
				}
			}
		}
	}
	return w.String()
}

func addExtraSpaceIfExist(str string) string {
	if str != "" {
		return " " + str
	}
	return ""
}

func checkOrPanic(err error) {
	if err != nil {
		panic(err)
	}
}

// check if value is nil
func isNil(value reflect.Value) bool {
	if value.Kind() != reflect.Ptr {
		return false
	}
	if value.Pointer() == 0 {
		return true
	}
	return false
}

type Alias struct {
	Expr string
	Name string
}

func BindVarStructure(dialect Dialector, field *FieldStructure, replacement string) string {
	var (
		reflectType = field.Type
		assigner    = field.Assigner
	)
	for reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}

	if assigner == nil {
		assigner = Assigners().Get(reflectType)
	}
	return BindVar(dialect, assigner, reflect.Indirect(reflect.New(reflectType)), replacement)
}

func BindVar(dialect Dialector, assigner Assigner, value interface{}, replacement string) string {
	if binder, ok := value.(ArgBinder); ok {
		return binder.DbBindVar(dialect, replacement)
	} else if assigner != nil {
		if binder, ok := assigner.(ArgBinder); ok {
			return binder.DbBindVar(dialect, replacement)
		}
	}
	return replacement
}

func SetZero(rvalue reflect.Value) {
	if reseter, ok := rvalue.Addr().Interface().(Reseter); ok {
		reseter.Reset()
	} else {
		rvalue.Set(reflect.Zero(rvalue.Type()))
	}
}

func SetNonZero(rvalue reflect.Value, value interface{}) {
	if rvalue.Kind() == reflect.Ptr {
		newValue := reflect.New(rvalue.Type().Elem())
		newValue.Elem().Set(reflect.ValueOf(value))
		rvalue.Set(newValue)
	} else {
		rvalue.Set(reflect.ValueOf(value))
	}
}

func IsManyResult(dst interface{}) (many bool) {
	typ := reflect.TypeOf(dst)
	for {
		switch typ.Kind() {
		case reflect.Ptr:
			typ = typ.Elem()
		case reflect.Slice, reflect.Chan, reflect.Array:
			return true
		default:
			return false
		}
	}
}

type AormNonFieldStructor interface {
	AormNonFieldStructor()
}

func StructTypeOf(value reflect.Type) (typ reflect.Type, many, ptr bool) {
	typ = value
	for {
		switch typ.Kind() {
		case reflect.Interface:
			typ = typ.Elem()
		case reflect.Ptr:
			ptr = true
			typ = typ.Elem()
		case reflect.Slice, reflect.Chan, reflect.Array:
			many = true
			ptr = false
			typ = typ.Elem()
		case reflect.Struct:
			if reflect.PtrTo(typ).Implements(reflect.TypeOf((*driver.Valuer)(nil)).Elem()) {
				if !reflect.PtrTo(typ).Implements(reflect.TypeOf((*AormNonFieldStructor)(nil)).Elem()) {
					// is a field type
					return nil, false, false
				}
			}
			return
		default:
			return nil, false, false
		}
	}
}

func StructTypeOfInterface(value interface{}) (typ reflect.Type, many, ptr bool) {
	var rtyp reflect.Type
	switch t := value.(type) {
	case reflect.Type:
		rtyp = t
	case reflect.Value:
		rtyp = t.Type()
	default:
		rtyp = reflect.TypeOf(t)
	}
	return StructTypeOf(rtyp)
}

func SenderOf(value reflect.Value) (send func(el reflect.Value)) {
	if value.Kind() != reflect.Ptr {
		panic("unaddressable value")
	}

	typ := value.Type().Elem()
	switch typ.Kind() {
	case reflect.Array:
		var i int
		value = value.Elem()
		value.Set(reflect.New(value.Type()))

		if typ.Elem().Kind() == reflect.Ptr {
			send = func(el reflect.Value) {
				value.Index(i).Set(el.Addr())
				i++
			}
		} else {
			send = func(el reflect.Value) {
				value.Index(i).Set(el)
				i++
			}
		}
		return
	case reflect.Slice:
		value.Elem().Set(reflect.MakeSlice(typ, 0, 0))
		value = value.Elem()

		if typ.Elem().Kind() == reflect.Ptr {
			send = func(el reflect.Value) {
				if el.Kind() == reflect.Ptr {
					value.Set(reflect.Append(value, el))
				} else {
					value.Set(reflect.Append(value, el.Addr()))
				}
			}
		} else {
			send = func(el reflect.Value) {
				if el.Kind() == reflect.Ptr {
					value.Set(reflect.Append(value, el.Elem()))
				} else {
					value.Set(reflect.Append(value, el))
				}
			}
		}
		return
	case reflect.Chan:
		value = value.Elem()
		if typ.Elem().Kind() == reflect.Ptr {
			send = func(el reflect.Value) {
				value.Send(el.Addr())
			}
		} else {
			send = func(el reflect.Value) {
				value.Send(el)
			}
		}
		return
	default:
		return
	}
}

func checkTruth(value interface{}) bool {
	if v, ok := value.(bool); ok && !v {
		return false
	}

	if v, ok := value.(string); ok {
		v = strings.ToLower(v)
		if v == "false" || v != "skip" {
			return false
		}
	}

	return true
}
