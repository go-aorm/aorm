package aorm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type Operation string

const (
	OpQuery  Operation = "select"
	OpCreate Operation = "create"
	OpUpdate Operation = "update"
	OpDelete Operation = "delete"
)

// Scope contain current operation's information when you perform any operation on the database
type Scope struct {
	Query
	Search          *search
	Value           interface{}
	db              *DB
	instanceID      string
	primaryKeyField *Field
	skipLeft        bool
	instance        *Instance
	selectAttrs     *[]string
	inlinePreloads  *InlinePreloads
	counter         bool
	ExecTime        time.Time
	modelStruct     *ModelStruct
	serial          *uint16
	Operation       Operation
}

func (scope *Scope) InlinePreloads() *InlinePreloads {
	return scope.inlinePreloads
}

// IndirectValue return scope's reflect value's indirect value
func (scope *Scope) IndirectValue() reflect.Value {
	return indirect(reflect.ValueOf(scope.Value))
}

// ResultSender return scope's value's result sender
func (scope *Scope) ResultSender() (dest reflect.Value, send func(el reflect.Value)) {
	dest = reflect.ValueOf(scope.Value)
	return dest, SenderOf(dest)
}

// New create a new Scope without search information
func (scope *Scope) New(value interface{}) *Scope {
	return &Scope{db: scope.NewDB(), Search: &search{}, Value: value, serial: scope.serial}
}

////////////////////////////////////////////////////////////////////////////////
// Scope DB
////////////////////////////////////////////////////////////////////////////////

// DB return scope's DB connection
func (scope *Scope) DB() *DB {
	return scope.db
}

// NewDB create a new DB without search information
func (scope *Scope) NewDB() *DB {
	if scope.db != nil {
		db := scope.db.clone()
		db.search = nil
		db.Val = nil
		return db
	}
	return nil
}

// SQLDB return *sql.DB
func (scope *Scope) SQLDB() SQLCommon {
	return scope.db.db
}

// Dialector get dialect
func (scope *Scope) Dialect() Dialector {
	return scope.db.dialect
}

// Quote used to quote string to escape them for database
func (scope *Scope) Quote(str string) string {
	if strings.Index(str, ".") != -1 {
		newStrs := []string{}
		for _, str := range strings.Split(str, ".") {
			newStrs = append(newStrs, Quote(scope.Dialect(), str))
		}
		return strings.Join(newStrs, ".")
	}

	return Quote(scope.Dialect(), str)
}

// Err add error to Scope
func (scope *Scope) Err(err error) error {
	if err != nil {
		if !IsRecordNotFoundError(err) && scope.modelStruct != nil && len(scope.modelStruct.UniqueIndexes) > 0 {
			err = scope.Dialect().DuplicateUniqueIndexError(scope.modelStruct.UniqueIndexes, scope.RealTableName(), err)
		}
		if scope.Query.Query != "" {
			err = NewQueryError(err, scope.Query, scope.db.dialect.BindVar)
		}
		scope.db.AddError(err)

		for _, cb := range scope.ErrorCallbacks() {
			cb(scope, err)
		}
	}
	return err
}

// HasError check if there are any error
func (scope *Scope) HasError() bool {
	return scope.db.Error != nil
}

// error return error
func (scope *Scope) Error() error {
	return scope.db.Error
}

// Log print log message
func (scope *Scope) Log(v ...interface{}) {
	scope.db.log(v...)
}

// SkipLeft skip remaining callbacks
func (scope *Scope) SkipLeft() {
	scope.skipLeft = true
}

// ValuesFields create value's fields from structFields
func (scope *Scope) ValuesFields(structFields []*StructField) []*Field {
	var (
		fields             = make([]*Field, len(structFields))
		indirectScopeValue = scope.IndirectValue()
		isStruct           = indirectScopeValue.Kind() == reflect.Struct
	)

	if isStruct {
		for i, structField := range structFields {
			fieldValue := indirectScopeValue
			for _, name := range structField.Names {
				if fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil() {
					fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
				}
				fieldValue = reflect.Indirect(fieldValue).FieldByName(name)
			}
			fields[i] = &Field{StructField: structField, Field: fieldValue, IsBlank: IsBlank(fieldValue)}
		}
	} else {
		for i, structField := range structFields {
			fields[i] = &Field{StructField: structField, IsBlank: true}
		}
	}
	return fields
}

// CreateFieldByName find `aorm.Field` with field name or db name
func (scope *Scope) FieldByName(name string) (field *Field, ok bool) {
	return scope.Instance().FieldByName(name)
}

// MustFieldByName find `aorm.Field` with field name or db name
func (scope *Scope) MustFieldByName(name string) (field *Field) {
	field, _ = scope.FieldByName(name)
	return
}

// HasPrimaryFields return if scope's has primary fields
func (scope *Scope) HasPrimaryFields() (ok bool) {
	return len(scope.Instance().Primary) > 0
}

// PrimaryFields return scope's primary fields
func (scope *Scope) PrimaryFields() (fields []*Field) {
	return scope.Instance().Primary
}

// PrimaryField return scope's main primary field, if defined more that one primary fields, will return the one having column name `id` or the first one
func (scope *Scope) PrimaryField() *Field {
	return scope.Instance().PrimaryField()
}

// PrimaryKeyDbName get main primary field's db name
func (scope *Scope) PrimaryKeyDbName() string {
	if field := scope.PrimaryField(); field != nil {
		return field.DBName
	}
	return ""
}

// PrimaryKeyZero check main primary field's value is blank or not
func (scope *Scope) PrimaryKeyZero() bool {
	field := scope.PrimaryField()
	return field == nil || field.IsBlank
}

// PrimaryKey get the primary key's value
func (scope *Scope) PrimaryKey() interface{} {
	if field := scope.PrimaryField(); field != nil && field.Field.IsValid() {
		return field.Field.Interface()
	}
	return nil
}

// HasColumn to check if has column
func (scope *Scope) HasColumn(column string) bool {
	for _, field := range scope.GetStructFields() {
		if field.IsNormal && (field.Name == column || field.DBName == column) {
			return true
		}
	}
	return false
}

// SetColumn to set the column's value, column could be field or field's name/dbname
func (scope *Scope) SetColumn(column interface{}, value interface{}) error {
	var updateAttrs = map[string]interface{}{}
	if attrs, ok := scope.InstanceGet("aorm:update_attrs"); ok {
		updateAttrs = attrs.(map[string]interface{})
		defer scope.InstanceSet("aorm:update_attrs", updateAttrs)
	}

	if field, ok := column.(*Field); ok {
		updateAttrs[field.DBName] = value
		return field.Set(value)
	} else if name, ok := column.(string); ok {
		var (
			fields           = scope.Instance()
			dbName           = ToDBName(name)
			mostMatchedField *Field
		)

		if field, ok := fields.FieldsMap[name]; ok {
			updateAttrs[field.DBName] = value
			return field.Set(value)
		}

		for _, field := range fields.Fields {
			if field.DBName == value {
				updateAttrs[field.DBName] = value
				return field.Set(value)
			}
			if (field.DBName == dbName) || (field.Name == name && mostMatchedField == nil) {
				mostMatchedField = field
			}
		}

		if mostMatchedField != nil {
			updateAttrs[mostMatchedField.DBName] = value
			return mostMatchedField.Set(value)
		}
	}
	return errors.New("could not convert column to field")
}

// CallMethod call scope value's Method, if it is a slice, will call its element's Method one by one
func (scope *Scope) CallMethod(methodName string) {
	if scope.Value == nil {
		return
	}

	if indirectScopeValue := scope.IndirectValue(); indirectScopeValue.Kind() == reflect.Slice {
		for i := 0; i < indirectScopeValue.Len(); i++ {
			scope.callMethod(methodName, indirectScopeValue.Index(i))
		}
	} else {
		scope.callMethod(methodName, indirectScopeValue)
	}
}

// SelectAttrs return selected attributes
func (scope *Scope) SelectAttrs() []string {
	if scope.selectAttrs == nil {
		attrs := []string{}
		for _, value := range scope.Search.selects {
			if str, ok := value.(string); ok {
				attrs = append(attrs, str)
			} else if strs, ok := value.([]string); ok {
				attrs = append(attrs, strs...)
			} else if strs, ok := value.([]interface{}); ok {
				for _, str := range strs {
					attrs = append(attrs, fmt.Sprintf("%v", str))
				}
			}
		}
		scope.selectAttrs = &attrs
	}
	return *scope.selectAttrs
}

// OmitAttrs return omitted attributes
func (scope *Scope) OmitAttrs() []string {
	return scope.Search.omits
}

// tableName return table name
func (scope *Scope) RealTableName() (name string) {
	var resolvers []TableNameResolver
	if value, ok := scope.InstanceGet(tableNameResolvers); ok {
		resolvers = value.([]TableNameResolver)
	}
	if value, ok := scope.Get(tableNameResolvers); ok {
		resolvers = append(resolvers, value.([]TableNameResolver)...)
	}

	var ok bool
	for _, resolver := range resolvers {
		if name, ok = resolver.TableName(scope.db.parent.singularTable, scope.Value); ok {
			return
		}
	}

	if tabler, ok := scope.Value.(TableNamer); ok {
		return tabler.TableName()
	}

	return scope.Struct().TableName(scope.db.Context, scope.db.singularTable)
}

func (scope *Scope) Context() context.Context {
	return scope.db.Context
}

// tableName return table name
func (scope *Scope) TableName() string {
	if scope.Search != nil {
		if scope.Search.tableAlias != "" {
			return scope.Search.tableAlias
		}
		if scope.Search.tableName != "" {
			return scope.Search.tableName
		}
	}

	return scope.RealTableName()
}

// QuotedTableName return quoted table name
func (scope *Scope) QuotedTableName() (name string) {
	return scope.Quote(scope.TableName())
}

// fromSql return from sql
func (scope *Scope) fromSql() (query string) {
	if s := scope.Search; s != nil {
		if s.from == "" {
			if s.tableName == "" {
				query = scope.Quote(scope.RealTableName())
			} else {
				query = s.tableName
			}
		} else {
			query = "(" + s.from + ")"
		}
		if s.tableAlias != "" {
			query += " AS " + scope.Quote(s.tableAlias)
		}
		return query
	}

	return scope.Quote(scope.RealTableName())
}

// CombinedConditionSql return combined condition sql
func (scope *Scope) CombinedConditionSql() string {
	joinSQL := scope.joinsSQL()
	whereSQL := scope.whereSQL()
	if scope.Search.raw {
		whereSQL = strings.TrimSuffix(strings.TrimPrefix(whereSQL, "WHERE ("), ")")
	}
	return joinSQL + whereSQL + scope.groupSQL() +
		scope.havingSQL() + scope.orderSQL() + scope.limitAndOffsetSQL()
}

// Raw set raw sql
func (scope *Scope) Raw(sql string) *Scope {
	scope.Query.Query = strings.Replace(sql, "$$$", "?", -1)
	return scope
}

// Exec perform generated SQL
func (scope *Scope) Exec() *Scope {
	scope.ExecResult()
	return scope
}

// ExecResult perform generated SQL and return result
func (scope *Scope) ExecResult() sql.Result {
	if !scope.HasError() {
		scope.ExecTime = NowFunc()
		defer scope.trace(scope.ExecTime)
		scope.log(LOG_EXEC)
		if result, err := scope.SQLDB().Exec(scope.Query.Query, scope.Query.Args...); scope.Err(err) == nil {
			if count, err := result.RowsAffected(); scope.Err(err) == nil {
				scope.db.RowsAffected = count
			}
			return result
		}
	}
	return nil
}

// Set set value by name
func (scope *Scope) Set(name string, value interface{}) *Scope {
	scope.db.InstantSet(name, value)
	return scope
}

// Get get setting by name
func (scope *Scope) Get(name string) (interface{}, bool) {
	return scope.db.Get(name)
}

// GetBool get boolean setting by name ou default
func (scope *Scope) GetBool(name string, defaul ...bool) bool {
	value, ok := scope.db.Get(name)
	if ok {
		return value.(bool)
	}
	for _, defaul := range defaul {
		return defaul
	}
	return false
}

// InstanceID get InstanceID for scope
func (scope *Scope) InstanceID() string {
	if scope.instanceID == "" {
		scope.instanceID = fmt.Sprintf("%v%v", &scope, &scope.db)
	}
	return scope.instanceID
}

// InstanceSet set instance setting for current operation, but not for operations in callbacks, like saving associations callback
func (scope *Scope) InstanceSet(name string, value interface{}) *Scope {
	return scope.Set(name+scope.InstanceID(), value)
}

// InstanceGet get instance setting from current operation
func (scope *Scope) InstanceGet(name string) (interface{}, bool) {
	return scope.Get(name + scope.InstanceID())
}

// InstanceGetBool get boolean instance setting from current operation
func (scope *Scope) InstanceGetBool(name string, defaul ...bool) bool {
	return scope.GetBool(name+scope.InstanceID(), defaul...)
}

// Begin start a transaction
func (scope *Scope) Begin() *Scope {
	if !scope.GetBool("aorm:disable_scope_transaction") {
		if db, ok := scope.SQLDB().(sqlDb); ok {
			if tx, err := db.Begin(); err == nil {
				scope.db.db = interface{}(tx).(SQLCommon)
				scope.InstanceSet("aorm:started_transaction", true)
			}
		}
	}
	return scope
}

// CommitOrRollback commit current transaction if no error happened, otherwise will rollback it
func (scope *Scope) CommitOrRollback() *Scope {
	if _, ok := scope.InstanceGet("aorm:started_transaction"); ok {
		if db, ok := scope.db.db.(sqlTx); ok {
			if scope.HasError() {
				db.Rollback()
			} else {
				scope.Err(db.Commit())
			}
			scope.db.db = scope.db.parent.db
		}
	}
	return scope
}

////////////////////////////////////////////////////////////////////////////////
// Private Methods For *aorm.Scope
////////////////////////////////////////////////////////////////////////////////

func (scope *Scope) callMethod(methodName string, reflectValue reflect.Value) {
	// Only get address from non-pointer
	if reflectValue.CanAddr() && reflectValue.Kind() != reflect.Ptr {
		reflectValue = reflectValue.Addr()
	}

	if methodValue := reflectValue.MethodByName(methodName); methodValue.IsValid() {
		switch method := methodValue.Interface().(type) {
		case func():
			method()
		case func(*Scope):
			method(scope)
		case func(*DB):
			newDB := scope.NewDB()
			method(newDB)
			scope.Err(newDB.Error)
		case func() error:
			scope.Err(method())
		case func(*Scope) error:
			scope.Err(method(scope))
		case func(*DB) error:
			newDB := scope.NewDB()
			scope.Err(method(newDB))
			scope.Err(newDB.Error)
		default:
			scope.Err(fmt.Errorf("unsupported function %v", methodName))
		}
	}
}

var (
	columnRegexp        = regexp.MustCompile("^[a-zA-Z\\d]+(\\.[a-zA-Z\\d]+)*$") // only match string like `name`, `users.name`
	isNumberRegexp      = regexp.MustCompile("^\\s*\\d+\\s*$")                   // match if string is number
	comparisonRegexp    = regexp.MustCompile("(?i) (=|<>|(>|<)(=?)|LIKE|IS|IN) ")
	countingQueryRegexp = regexp.MustCompile("(?i)^count(.+)$")
)

func (scope *Scope) quoteIfPossible(str string) string {
	if columnRegexp.MatchString(str) {
		return scope.Quote(str)
	}
	return str
}

// call after field method callbacks
func (scope *Scope) afterScanCallback(scannerFields map[int]*Field, disableScanField map[int]bool) {
	if !scope.HasError() && scope.Value != nil {
		if scope.DB().IsEnabledAfterScanCallback(scope.Value) {
			scopeValue := reflect.ValueOf(scope)
			for index, field := range scannerFields {
				// if not is nill and if calbacks enabled for field type
				if StructFieldMethodCallbacks.IsEnabledFieldType(field.Field.Type()) {
					// not disabled on scan
					if _, ok := disableScanField[index]; !ok {
						if !isNil(field.Field) {
							reflectValue := field.Field.Addr()
							field.CallMethodCallback("AfterScan", reflectValue, scopeValue)
						}
					}
				}
			}
		}
	}
}

func (scope *Scope) defaultColumnValue(result interface{}, column string) interface{} {
	return nil
}

func (scope *Scope) primaryCondition(value interface{}) string {
	return fmt.Sprintf("(%v.%v = %v)", scope.QuotedTableName(), scope.Quote(scope.PrimaryKeyDbName()), value)
}

func (scope *Scope) inlineCondition(values ...interface{}) *Scope {
	if len(values) > 0 {
		scope.Search.Where(values[0], values[1:]...)
	}
	return scope
}

func (scope *Scope) callCallbacks(funcs []*func(s *Scope)) *Scope {
	for _, f := range funcs {
		(*f)(scope)
		if scope.skipLeft || scope.HasError() {
			break
		}
	}
	scope.db.Query = &scope.Query
	return scope
}

func (scope *Scope) updatedAttrsWithValues(value interface{}) (results map[string]interface{}, hasUpdate bool) {
	var storeBlankField bool
	if v, ok := scope.Get(StoreBlankField); ok {
		storeBlankField = v.(bool)
	}
	if scope.IndirectValue().Kind() != reflect.Struct {
		return convertInterfaceToMap(scope, value, false, storeBlankField), true
	}

	results = map[string]interface{}{}

	for key, value := range convertInterfaceToMap(scope, value, true, storeBlankField) {
		if field, ok := scope.FieldByName(key); ok && scope.changeableField(field) {
			if _, ok := value.(*Query); ok {
				hasUpdate = true
				results[field.DBName] = value
			} else {
				err := field.Set(value)
				if field.IsNormal {
					hasUpdate = true
					if err == ErrUnaddressable {
						results[field.DBName] = value
					} else {
						results[field.DBName] = field.Field.Interface()
					}
				}
			}
		}
	}
	return
}

func (scope *Scope) row() *sql.Row {
	defer scope.trace(NowFunc())

	result := &RowQueryResult{}
	scope.InstanceSet("row_query_result", result)
	scope.callCallbacks(scope.db.parent.callbacks.rowQueries)

	return result.Row
}

func (scope *Scope) rows() (*sql.Rows, error) {
	defer scope.trace(NowFunc())

	result := &RowsQueryResult{}
	scope.InstanceSet("row_query_result", result)
	scope.callCallbacks(scope.db.parent.callbacks.rowQueries)

	return result.Rows, result.Error
}

func (scope *Scope) initialize() *Scope {
	for _, clause := range scope.Search.whereConditions {
		scope.updatedAttrsWithValues(clause.Query)
	}
	scope.updatedAttrsWithValues(scope.Search.initAttrs)
	scope.updatedAttrsWithValues(scope.Search.assignAttrs)
	return scope
}

func (scope *Scope) isQueryForColumn(query interface{}, column string) bool {
	queryStr := strings.ToLower(fmt.Sprint(query))
	if queryStr == column {
		return true
	}

	if strings.HasSuffix(queryStr, "as "+column) {
		return true
	}

	if strings.HasSuffix(queryStr, "as "+scope.Quote(column)) {
		return true
	}

	return false
}

func (scope *Scope) pluck(column string, value interface{}) *Scope {
	dest := reflect.Indirect(reflect.ValueOf(value))
	if dest.Kind() != reflect.Slice {
		scope.Err(fmt.Errorf("results should be a slice, not %s", dest.Kind()))
		return scope
	}

	if query, ok := scope.Search.selects["query"]; !ok || !scope.isQueryForColumn(query, column) {
		scope.Search.Select(column)
	}

	rows, err := scope.rows()
	if scope.Err(err) == nil {
		defer rows.Close()
		for rows.Next() {
			elem := reflect.New(dest.Type().Elem()).Interface()
			scope.Err(rows.Scan(elem))
			dest.Set(reflect.Append(dest, reflect.ValueOf(elem).Elem()))
		}

		if err := rows.Err(); err != nil {
			scope.Err(err)
		}
	}
	return scope
}

func (scope *Scope) pluckFirst(column string, value interface{}) *Scope {
	if query, ok := scope.Search.selects["query"]; !ok || !scope.isQueryForColumn(query, column) {
		scope.Search.Select(column)
	}

	scope.Search.Limit(1)
	rows, err := scope.rows()

	if scope.Err(err) == nil {
		defer rows.Close()
		var has bool
		for rows.Next() {
			has = true
			scope.Err(rows.Scan(value))
			if err := rows.Err(); err != nil {
				scope.Err(err)
			}
		}
		if !has {
			scope.Err(ErrRecordNotFound)
		}
	}
	return scope
}

func (scope *Scope) count(value interface{}) *Scope {
	if query, ok := scope.Search.selects["query"]; !ok || !countingQueryRegexp.MatchString(fmt.Sprint(query)) {
		if len(scope.Search.group) != 0 {
			scope.Search.Select("count(*) FROM ( SELECT count(*) as name ")
			scope.Search.group += " ) AS count_table"
		} else {
			scope.Search.Select("count(*)")
		}
	}
	scope.Search.ignoreOrderQuery = true
	scope.counter = true
	scope.Err(scope.row().Scan(value))
	return scope
}

func (scope *Scope) exists() (ok bool, err error) {
	scope.counter = true
	scope.Search.Select("TRUE")
	scope.Search.Limit(1)
	row := scope.row()
	if err = scope.Error(); err != nil || row == nil {
		if IsRecordNotFoundError(err) {
			err = nil
		}
		return
	}
	err = row.Scan(&ok)
	return
}

func (scope *Scope) typeName() string {
	typ := scope.IndirectValue().Type()

	for typ.Kind() == reflect.Slice || typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	return typ.Name()
}

// trace print sql log
func (scope *Scope) trace(t time.Time) {
	if len(scope.Query.Query) > 0 {
		scope.db.slog(scope.Query.Query, t, scope.Query.Args...)
	}
}

func (scope *Scope) changeableField(field *Field) bool {
	if field.IsReadOnly {
		return false
	}
	if selectAttrs := scope.SelectAttrs(); len(selectAttrs) > 0 {
		for _, attr := range selectAttrs {
			if field.Name == attr || field.DBName == attr {
				return true
			}
		}
		return false
	}

	for _, attr := range scope.OmitAttrs() {
		if field.Name == attr || field.DBName == attr {
			return false
		}
	}

	return true
}

func (scope *Scope) related(value interface{}, foreignKeys ...string) *Scope {
	toScope := scope.db.NewScope(value)
	db := *toScope.db
	db.search = db.search.resetConditions()
	toScope.Search = &search{db: &db}
	toScope.db = &db

	tx := toScope.db.Set("aorm:association:source", scope.Value)

	if onRelated, ok := value.(OnRelated); ok {
		tx = onRelated.AormOnRelated(scope, toScope, tx)
	}

	var prepare = func(fromField *Field) {
		if onRelated, ok := value.(OnRelatedField); ok {
			tx = onRelated.AormOnRelatedField(scope, toScope, tx, fromField)
		}

		for _, cb := range toScope.Struct().BeforeRelatedCallbacks {
			tx = cb(scope, toScope, tx, fromField)
		}
	}

	for _, foreignKey := range append(foreignKeys, toScope.typeName()+"Id", scope.typeName()+"Id") {
		fromField, _ := scope.FieldByName(foreignKey)
		toField, _ := toScope.FieldByName(foreignKey)

		if fromField != nil {
			if relationship := fromField.Relationship; relationship != nil {
				if relationship.Kind == "many_to_many" {
					prepare(fromField)
					joinTableHandler := relationship.JoinTableHandler
					scope.Err(joinTableHandler.JoinWith(joinTableHandler, tx, scope.Value).Find(value).Error)
				} else if relationship.Kind == "belongs_to" {
					if relationship.InstanceToRelatedID(scope.instance).IsZero() {
						return scope
					}
					prepare(fromField)
					for idx, foreignKey := range relationship.ForeignDBNames {
						if field, ok := scope.FieldByName(foreignKey); ok {
							tx = tx.Where(fmt.Sprintf("%v.%v = ?", toScope.QuotedTableName(), scope.Quote(relationship.AssociationForeignDBNames[idx])), field.Field.Interface())
						}
					}
					scope.Err(tx.Find(value).Error)
				} else if relationship.Kind == "has_many" || relationship.Kind == "has_one" {
					prepare(fromField)
					for idx, foreignKey := range relationship.ForeignDBNames {
						if field, ok := scope.FieldByName(relationship.AssociationForeignDBNames[idx]); ok {
							tx = tx.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
						}
					}

					if relationship.PolymorphicType != "" {
						tx = tx.Where(fmt.Sprintf("%v = ?", scope.Quote(relationship.PolymorphicDBName)),
							relationship.PolymorphicValue(scope.db.Context, scope.db.singularTable))
					}
					scope.Err(tx.Find(value).Error)
				}
			} else {
				prepare(fromField)
				sql := fmt.Sprintf("%v = ?", scope.Quote(toScope.PrimaryKeyDbName()))
				scope.Err(tx.Where(sql, fromField.Field.Interface()).Find(value).Error)
			}
			return scope
		} else if toField != nil {
			sql := fmt.Sprintf("%v = ?", scope.Quote(toField.DBName))
			scope.Err(tx.Where(sql, scope.PrimaryKey()).Find(value).Error)
			return scope
		}
	}

	scope.Err(fmt.Errorf("invalid association %v", foreignKeys))
	return scope
}

// getTableOptions return the table options string or an empty string if the table options does not exist
func (scope *Scope) getTableOptions() string {
	tableOptions, ok := scope.Get("aorm:table_options")
	if !ok {
		return ""
	}
	return " " + tableOptions.(string)
}

func (scope *Scope) getColumnAsArray(columns []string, values ...interface{}) (results [][]interface{}) {
	for _, value := range values {
		indirectValue := indirect(reflect.ValueOf(value))

		switch indirectValue.Kind() {
		case reflect.Slice:
			for i := 0; i < indirectValue.Len(); i++ {
				var result []interface{}
				var object = indirect(indirectValue.Index(i))
				var hasValue = false
				for _, column := range columns {
					field := object.FieldByName(column)
					if hasValue || !IsBlank(field) {
						hasValue = true
					}
					result = append(result, field.Interface())
				}

				if hasValue {
					results = append(results, result)
				}
			}
		case reflect.Struct:
			var result []interface{}
			var hasValue = false
			for _, column := range columns {
				field := indirectValue.FieldByName(column)
				if hasValue || !IsBlank(field) {
					hasValue = true
				}
				result = append(result, field.Interface())
			}

			if hasValue {
				results = append(results, result)
			}
		}
	}

	return
}

func (scope *Scope) ScopeOfField(fieldName string) *Scope {
	indirectScopeValue := scope.IndirectValue()

	if fieldStruct, ok := scope.Struct().FieldsByName[fieldName]; ok {
		switch indirectScopeValue.Kind() {
		case reflect.Slice:
			fieldType := fieldStruct.Struct.Type
			if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}

			resultsMap := map[interface{}]bool{}
			results := reflect.New(reflect.SliceOf(reflect.PtrTo(fieldType))).Elem()

			for i := 0; i < indirectScopeValue.Len(); i++ {
				result := indirect(indirect(indirectScopeValue.Index(i)).FieldByName(fieldName))

				if result.Kind() == reflect.Slice {
					for j := 0; j < result.Len(); j++ {
						if elem := result.Index(j); elem.CanAddr() && resultsMap[elem.Addr()] != true {
							resultsMap[elem.Addr()] = true
							results = reflect.Append(results, elem.Addr())
						}
					}
				} else if result.CanAddr() && resultsMap[result.Addr()] != true {
					resultsMap[result.Addr()] = true
					results = reflect.Append(results, result.Addr())
				}
			}
			return scope.New(results.Interface())
		case reflect.Struct:
			if field := indirectScopeValue.FieldByIndex(fieldStruct.StructIndex); field.CanAddr() {
				if field.Kind() == reflect.Ptr {
					if isNil(field) {
						value := reflect.New(field.Type().Elem()).Interface()
						return scope.New(value)
					}
					return scope.New(field.Interface())
				}
				return scope.New(field.Addr().Interface())
			}
		}
	}
	return nil
}

func (scope *Scope) hasConditions() bool {
	return !scope.PrimaryKeyZero() ||
		len(scope.Search.whereConditions) > 0 ||
		len(scope.Search.orConditions) > 0 ||
		len(scope.Search.notConditions) > 0
}

func (s *Scope) SetVirtualField(fieldName string, value interface{}, options ...map[interface{}]interface{}) (vf *VirtualField) {
	if value == nil {
		value = &VirtualFieldValue{}
	}
	vf = s.Struct().SetVirtualField(fieldName, value)
	for _, options := range options {
		for key, value := range options {
			vf.Options[key] = value
		}
	}
	return vf
}

func (s *Scope) GetVirtualField(fieldName string) *VirtualField {
	return s.Struct().GetVirtualField(fieldName)
}

func (s *Scope) runQueryRows() (rows *sql.Rows) {
	var err error
	s.log(LOG_QUERY)
	if rows, err = s.SQLDB().Query(s.Query.Query, s.Query.Args...); err != nil {
		s.Err(err)
		return nil
	}
	return
}

func (s *Scope) runQueryRow() (row *sql.Row) {
	s.log(LOG_QUERY)
	return s.SQLDB().QueryRow(s.Query.Query, s.Query.Args...)
}

func (s *Scope) execQuery() (result sql.Result, err error) {
	s.log(LOG_QUERY)
	return s.SQLDB().Exec(s.Query.Query, s.Query.Args...)
}

func (s *Scope) Loggers(set ...bool) (sl *ScopeLoggers, ok bool) {
	key := scopeLoggerKey(s.RealTableName())
	if cbsv, ok := s.Get(key); ok {
		return cbsv.(*ScopeLoggers), true
	} else if len(set) > 0 && set[0] {
		sl = &ScopeLoggers{}
		s.Set(key, sl)
	}
	return
}

func (s *Scope) MustLoggers(set ...bool) (sl *ScopeLoggers) {
	sl, _ = s.Loggers(set...)
	return
}

func (s *Scope) log(action string) *Scope {
	if sl, ok := s.Loggers(); ok {
		sl.Call(action, s)
	}
	if _, sl, ok := s.db.Loggers(s.RealTableName()); ok {
		sl.Call(action, s)
	}
	DefaultLogger.Call(action, s)
	return s
}

func (s *Scope) ErrorCallbacks() (callbacks []func(scope *Scope, err error)) {
	if cbsv, ok := s.Get("aorm:scope_error_callbacks"); ok {
		callbacks = cbsv.([]func(scope *Scope, err error))
	}
	return
}
