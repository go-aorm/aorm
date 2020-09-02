package aorm

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/moisespsena-go/tracederror"
	"github.com/pkg/errors"

	"github.com/moisespsena-go/logging"
)

// DB contains information for current db connection
type DB struct {
	Val          interface{}
	Error        error
	RowsAffected int64

	// single db
	db                SQLCommon
	blockGlobalUpdate bool
	logMode           int
	logger            logging.Logger
	search            *search
	values            map[interface{}]interface{}
	path              []interface{}

	// global db
	parent             *DB
	callbacks          *Callback
	dialect            Dialector
	singularTable      bool
	assigners          *AssignerRegistrator
	modelStructStorage *ModelStructStorage
	Context            context.Context
	noExec             bool

	Query       *Query
	modelStruct *ModelStruct

	migrator *Migrator
}

func NewDB(dialect ...Dialector) (db *DB) {
	modelStructStorage := NewModelStructStorage()
	modelStructStorage.GetAssigner = func(typ reflect.Type) Assigner {
		return db.GetAssigner(typ)
	}

	db = &DB{
		logger:             log,
		values:             map[interface{}]interface{}{},
		callbacks:          DefaultCallback,
		Context:            context.Background(),
		modelStructStorage: modelStructStorage,
	}
	db.parent = db
	for _, db.dialect = range dialect {
	}
	return db
}

// Open initialize a new db connection, need to import driver first, e.g:
//
//     import _ "github.com/go-sql-driver/mysql"
//     func main() {
//       db, err := aorm.Open("mysql", "user:password@/dbname?charset=utf8&parseTime=True&loc=Local")
//     }
// AORM has wrapped some drivers, for easier to remember driver's import path, so you could import the mysql driver with
//    import _ "github.com/moisespsena-go/aorm/dialects/mysql"
//    // import _ "github.com/moisespsena-go/aorm/dialects/postgres"
//    // import _ "github.com/moisespsena-go/aorm/dialects/sqlite"
//    // import _ "github.com/moisespsena-go/aorm/dialects/mssql"
func Open(dialect string, args ...interface{}) (db *DB, err error) {
	if len(args) == 0 {
		err = errors.New("invalid database source")
		return nil, err
	}
	var source string
	var dbSQL SQLCommon
	var ownDbSQL bool

	switch value := args[0].(type) {
	case string:
		var driver = dialect
		if len(args) == 1 {
			source = value
		} else if len(args) >= 2 {
			driver = value
			source = args[1].(string)
		}
		if dbSQL, err = sql.Open(driver, source); err != nil {
			return nil, errors.Wrap(err, "open")
		}
		ownDbSQL = true
	case SQLCommon:
		dbSQL = value
		ownDbSQL = false
	default:
		return nil, errors.New(fmt.Sprintf("invalid database source: %v is not a valid type", value))
	}

	db = NewDB()
	db.db = dbSQL
	db.dialect = newDialect(dialect, dbSQL)

	if err != nil {
		return
	}
	// Send a ping to make sure the database connection is alive.
	if d, ok := dbSQL.(*sql.DB); ok {
		if err = d.Ping(); err != nil && ownDbSQL {
			d.Close()
		}

	}
	return
}

// New clone a new db connection without search conditions
func (s *DB) New() *DB {
	clone := s.clone()
	clone.search = nil
	clone.Val = nil
	return clone
}

// Close close current db connection.  If database connection is not an io.Closer, returns an error.
func (s *DB) Close() error {
	if db, ok := s.parent.db.(io.Closer); ok {
		return db.Close()
	}
	return errors.New("can'T close current db")
}

// DB get `*sql.DB` from current connection
// If the underlying database connection is not a *sql.DB, returns nil
func (s *DB) DB() *sql.DB {
	db, _ := s.db.(*sql.DB)
	return db
}

// CommonDB return the underlying `*sql.DB` or `*sql.Tx` instance, mainly intended to allow coexistence with legacy non-AORM code.
func (s *DB) CommonDB() SQLCommon {
	return s.db
}

// SetCommonDB set the underlying `*sql.DB` or `*sql.Tx` instance, mainly intended to allow coexistence with legacy non-AORM code.
func (s *DB) SetCommonDB(db SQLCommon) *DB {
	clone := s.clone()
	clone.db = db
	return clone
}

// Dialector get dialect
func (s *DB) Dialect() Dialector {
	return s.dialect
}

// Callback return `Callbacks` container, you could add/change/delete callbacks with it
//     db.Callback().Create().Register("update_created_at", updateCreated)
// Refer https://jinzhu.github.io/gorm/development.html#callbacks
func (s *DB) Callback() *Callback {
	s.parent.callbacks = s.parent.callbacks.clone()
	return s.parent.callbacks
}

// SetLogger replace default logger
func (s *DB) SetLogger(log logging.Logger) {
	s.logger = log
}

// LogMode set log mode, `true` for detailed logs, `false` for no log, default, will only print error logs
func (s *DB) LogMode(enable bool) *DB {
	if enable {
		s.logMode = 2
	} else {
		s.logMode = 1
	}
	return s
}

// BlockGlobalUpdate if true, generates an error on update/delete without where clause.
// This is to prevent eventual error with empty objects updates/deletions
func (s *DB) BlockGlobalUpdate(enable bool) *DB {
	s.blockGlobalUpdate = enable
	return s
}

// HasBlockGlobalUpdate return state of block
func (s *DB) HasBlockGlobalUpdate() bool {
	return s.blockGlobalUpdate
}

// SingularTable use singular table by default
func (s *DB) SingularTable(enable bool) {
	s.parent.singularTable = enable
}

// IsSingularTable returns if use singular table
func (s *DB) IsSingularTable() bool {
	return s.singularTable
}

// NewScope create a scope for current operation
func (s *DB) NewScope(value interface{}) *Scope {
	return s.NewModelScope(s.modelStruct, value)
}

// NewModelScope create a scope for current operation with ModelStruct
func (s *DB) NewModelScope(model *ModelStruct, value interface{}) *Scope {
	dbClone := s.clone()
	dbClone.Val = value
	dbClone.modelStruct = model

	if model == nil {
		model = StructOf(value)
	}

	var serial uint16
	return &Scope{
		db:          dbClone,
		Search:      dbClone.search.clone(),
		Value:       value,
		serial:      &serial,
		modelStruct: model,
	}
}

// QueryExpr returns the query as Query object
func (s *DB) QueryExpr() *Query {
	scope := s.NewScope(s.Val)
	scope.InstanceSet("skip_bindvar", true)
	scope.prepareQuerySQL()
	return &scope.Query
}

// SubQuery returns the query as sub query
func (s *DB) SubQuery() *Query {
	scope := s.NewScope(s.Val)
	scope.InstanceSet("skip_bindvar", true)
	scope.prepareQuerySQL()

	return Expr(fmt.Sprintf("(%v)", scope.Query.Query), scope.Query.Args...)
}

// Where return a new relation, filter records with given conditions, accepts `map`, `struct` or `string` as conditions, refer http://jinzhu.github.io/gorm/crud.html#query
func (s *DB) Where(query interface{}, args ...interface{}) *DB {
	return s.clone().search.Where(query, args...).db
}

// Or filter records that match before conditions or this one, similar to `Where`
func (s *DB) Or(query interface{}, args ...interface{}) *DB {
	return s.clone().search.Or(query, args...).db
}

// Not filter records that don'T match current conditions, similar to `Where`
func (s *DB) Not(query interface{}, args ...interface{}) *DB {
	return s.clone().search.Not(query, args...).db
}

// Limit specify the number of records to be retrieved
func (s *DB) Limit(limit interface{}) *DB {
	return s.clone().search.Limit(limit).db
}

// Offset specify the number of records to skip before starting to return the records
func (s *DB) Offset(offset interface{}) *DB {
	return s.clone().search.Offset(offset).db
}

// Order specify order when retrieve records from database, set reorder to `true` to overwrite defined conditions
//     db.Order("name DESC")
//     db.Order("name DESC", true) // reorder
//     db.Order(aorm.Expr("name = ? DESC", "first")) // sql expression
func (s *DB) Order(value interface{}, reorder ...bool) *DB {
	return s.clone().search.Order(value, reorder...).db
}

// HasOrder returns if order has be specified
func (s *DB) HasOrder() bool {
	return s.search.orders != nil
}

// Select specify fields that you want to retrieve from database when querying, by default, will select all fields;
// When creating/updating, specify fields that you want to save to database
func (s *DB) Select(query interface{}, args ...interface{}) *DB {
	return s.clone().search.Select(query, args...).db
}

// ExtraSelect specify extra fields that you want to retrieve from database when querying
func (s *DB) ExtraSelect(key string, values []interface{}, query interface{}, args ...interface{}) *DB {
	return s.clone().search.ExtraSelect(key, values, query, args...).db
}

// ExtraSelectCalback specify extra select callbacks that you want to retrieve from database when querying
func (s *DB) ExtraSelectCallback(f ...func(recorde interface{}, data map[string]*ExtraResult)) *DB {
	return s.clone().search.ExtraSelectCallback(f...).db
}

// ExtraSelectFields specify extra fields that you want to retrieve from database when querying
func (s *DB) ExtraSelectFields(key string, value interface{}, names []string, callback func(scope *Scope, record interface{}), query interface{}, args ...interface{}) *DB {
	modelStruct := s.NewScope(value).Struct()
	fields := make([]*StructField, len(names))
	for i, name := range names {
		if f, ok := modelStruct.FieldsByName[name]; ok {
			fields[i] = f
		} else {
			panic(s.AddError(fmt.Errorf("Invalid field %q", name)))
		}
	}
	return s.clone().search.ExtraSelectFields(key, value, fields, callback, query, args...).db
}

// ExtraSelectFieldsSetter specify extra fields that you want to retrieve from database when querying
func (s *DB) ExtraSelectFieldsSetter(key string, setter ExtraSelectFieldsSetter, structFields []*StructField, query interface{}, args ...interface{}) *DB {
	return s.clone().search.ExtraSelectFields(key, setter, structFields, nil, query, args...).db
}

// Omit specify fields that you want to ignore when saving to database for creating, updating
func (s *DB) Omit(columns ...string) *DB {
	return s.clone().search.Omit(columns...).db
}

// Group specify the group method on the find
func (s *DB) Group(query string) *DB {
	return s.clone().search.Group(query).db
}

// Having specify HAVING conditions for GROUP BY
func (s *DB) Having(query interface{}, values ...interface{}) *DB {
	return s.clone().search.Having(query, values...).db
}

// Joins specify Joins conditions
//     db.Joins("JOIN emails ON emails.user_id = users.id AND emails.email = ?", "jinzhu@example.org").Find(&user)
func (s *DB) Joins(query string, args ...interface{}) *DB {
	return s.clone().search.Joins(query, args...).db
}

// Join specify Join field conditions
//     db.Model(&User{}).Join("INNER", "Email")
//     // INNER JOIN emails ON emails.user_id = user.id
func (s *DB) Join(fieldName, mode string, handler func()) *DB {
	panic("not implemented")
}

// Scopes pass current database connection to arguments `func(*DB) *DB`, which could be used to add conditions dynamically
//     func AmountGreaterThan1000(db *aorm.DB) *aorm.DB {
//         return db.Where("amount > ?", 1000)
//     }
//
//     func OrderStatus(status []string) func (db *aorm.DB) *aorm.DB {
//         return func (db *aorm.DB) *aorm.DB {
//             return db.Scopes(AmountGreaterThan1000).Where("status in (?)", status)
//         }
//     }
//
//     db.Scopes(AmountGreaterThan1000, OrderStatus([]string{"paid", "shipped"})).Find(&orders)
// Refer https://jinzhu.github.io/gorm/crud.html#scopes
func (s *DB) Scopes(funcs ...func(*DB) *DB) *DB {
	for _, f := range funcs {
		s = f(s)
	}
	return s
}

// Unscoped return all record including deleted record, refer Soft Delete https://jinzhu.github.io/gorm/crud.html#soft-delete
func (s *DB) Unscoped() *DB {
	return s.clone().search.unscoped().db
}

// Attrs initialize struct with argument if record not found with `FirstOrInit` https://jinzhu.github.io/gorm/crud.html#firstorinit or `FirstOrCreate` https://jinzhu.github.io/gorm/crud.html#firstorcreate
func (s *DB) Attrs(attrs ...interface{}) *DB {
	return s.clone().search.Attrs(attrs...).db
}

// Assigner assign result with argument regardless it is found or not with `FirstOrInit` https://jinzhu.github.io/gorm/crud.html#firstorinit or `FirstOrCreate` https://jinzhu.github.io/gorm/crud.html#firstorcreate
func (s *DB) Assign(attrs ...interface{}) *DB {
	return s.clone().search.Assign(attrs...).db
}

// First find first record that match given conditions, order by primary key
func (s *DB) First(out interface{}, where ...interface{}) *DB {
	newScope := s.NewScope(out)
	newScope.Search.Limit(1)
	return newScope.Set("aorm:order_by_primary_key", "ASC").
		inlineCondition(where...).callCallbacks(s.parent.callbacks.queries).db
}

// Take return a record that match given conditions, the order will depend on the database implementation
func (s *DB) Take(out interface{}, where ...interface{}) *DB {
	newScope := s.NewScope(out)
	newScope.Search.Limit(1)
	return newScope.inlineCondition(where...).callCallbacks(s.parent.callbacks.queries).db
}

// Last find last record that match given conditions, order by primary key
func (s *DB) Last(out interface{}, where ...interface{}) *DB {
	newScope := s.NewScope(out)
	newScope.Search.Limit(1)
	return newScope.Set("aorm:order_by_primary_key", "DESC").
		inlineCondition(where...).callCallbacks(s.parent.callbacks.queries).db
}

// Find find records that match given conditions
func (s *DB) Find(out interface{}, where ...interface{}) *DB {
	if t := indirectType(reflect.TypeOf(out)); t.Kind() == reflect.Array {
		s = s.Limit(t.Len())
	}
	return s.NewScope(out).inlineCondition(where...).callCallbacks(s.parent.callbacks.queries).db
}

// Scan scan value to a struct
func (s *DB) Scan(dest interface{}) *DB {
	v := s.Val
	if v == nil {
		v = dest
	}
	return s.NewScope(v).Set("aorm:query_destination", dest).callCallbacks(s.parent.callbacks.queries).db
}

// Row return `*sql.Row` with given conditions
func (s *DB) Row() *sql.Row {
	return s.NewScope(s.Val).row()
}

// Rows return `*sql.Rows` with given conditions
func (s *DB) Rows() (*sql.Rows, error) {
	return s.NewScope(s.Val).rows()
}

// ScanRows scan `*sql.Rows` to give struct
func (s *DB) ScanRows(rows *sql.Rows, result interface{}) error {
	var (
		scope        = s.NewScope(result)
		clone        = scope.db
		columns, err = rows.Columns()
	)

	if clone.AddError(err) == nil {
		scope.scan(rows, columns, scope.Instance().Fields, result)
	}

	return clone.Error
}

// Pluck used to query single column from a model as a map
//     var ages []int64
//     db.Find(&users).Pluck("age", &ages)
func (s *DB) Pluck(column string, value interface{}) *DB {
	return s.NewScope(s.Val).pluck(column, value).db
}

// PluckFirst used to query single column from a first model result as a map
//     var createAt time.Time
//     db.Model(&User{}).Pluck("created_at", &createdAt)
func (s *DB) PluckFirst(column string, value interface{}) *DB {
	return s.NewScope(s.Val).pluckFirst(column, value).db
}

// Exists used to query single column from a first model result as a map
//     var createAt time.Time
//     db.Model(&User{}).Exists()
func (s *DB) Exists(where ...interface{}) (ok bool, err error) {
	return s.NewScope(s.Val).Set(OptKeySkipPreload, true).inlineCondition(where...).exists()
}

// ExistsValue used to query single column from a first model result as a map
//     var createAt time.Time
//     db.ExistsValue(&User{Name:"user name"})
func (s *DB) ExistsValue(value interface{}, where ...interface{}) (ok bool, err error) {
	return s.NewScope(value).inlineCondition(where...).exists()
}

// Count get how many records for a model
func (s *DB) Count(value interface{}) *DB {
	return s.NewScope(s.Val).count(value).db
}

// Related get related associations
func (s *DB) Related(value interface{}, foreignFieldNames ...string) *DB {
	return s.RelatedModel(StructOf(value), value, foreignFieldNames...)
}

// RelatedModel get related associations with model struct
func (s *DB) RelatedModel(model *ModelStruct, value interface{}, foreignFieldNames ...string) *DB {
	return s.NewScope(s.Val).related(model, value, foreignFieldNames...).db
}

// FirstOrInit find first matched record or initialize a new one with given conditions (only works with struct, map conditions)
// https://jinzhu.github.io/gorm/crud.html#firstorinit
func (s *DB) FirstOrInit(out interface{}, where ...interface{}) *DB {
	c := s.clone()
	if result := c.First(out, where...); result.Error != nil {
		if !result.RecordNotFound() {
			return result
		}
		c.NewScope(out).inlineCondition(where...).initialize()
	} else {
		c.NewScope(out).updatedAttrsWithValues(c.search.assignAttrs)
	}
	return c
}

// FirstOrCreate find first matched record or create a new one with given conditions (only works with struct, map conditions)
// https://jinzhu.github.io/gorm/crud.html#firstorcreate
func (s *DB) FirstOrCreate(out interface{}, where ...interface{}) *DB {
	c := s.clone()
	if result := s.First(out, where...); result.Error != nil {
		if !result.RecordNotFound() {
			return result
		}
		return c.NewScope(out).inlineCondition(where...).initialize().callCallbacks(c.parent.callbacks.creates).db
	} else if len(c.search.assignAttrs) > 0 {
		return c.NewScope(out).InstanceSet("aorm:update_interface", c.search.assignAttrs).callCallbacks(c.parent.callbacks.updates).db
	}
	return c
}

// FirstOrCreate find first matched record or create a new one with given conditions (only works with struct, map conditions)
// https://jinzhu.github.io/gorm/crud.html#firstorcreate
func (s *DB) firstOrCreate(out interface{}, where ...interface{}) *DB {
	c := s.clone()
	if result := s.First(out, where...); result.Error != nil {
		if !result.RecordNotFound() {
			return result
		}
		return c.NewScope(out).inlineCondition(where...).initialize().callCallbacks(c.parent.callbacks.creates).db
	}
	return c
}

// Update update attributes with callbacks, refer: https://jinzhu.github.io/gorm/crud.html#update
func (s *DB) Update(attrs ...interface{}) *DB {
	return s.Updates(toSearchableMap(attrs...), true)
}

// Updates update attributes with callbacks, refer: https://jinzhu.github.io/gorm/crud.html#update
func (s *DB) Updates(values interface{}, ignoreProtectedAttrs ...bool) *DB {
	return s.NewScope(s.Val).
		Set("aorm:ignore_protected_attrs", len(ignoreProtectedAttrs) > 0).
		InstanceSet("aorm:update_interface", values).
		callCallbacks(s.parent.callbacks.updates).db
}

// UpdateColumn update attributes without callbacks, refer: https://jinzhu.github.io/gorm/crud.html#update
func (s *DB) UpdateColumn(attrs ...interface{}) *DB {
	return s.UpdateColumns(toSearchableMap(attrs...))
}

// UpdateColumns update attributes without callbacks, refer: https://jinzhu.github.io/gorm/crud.html#update
func (s *DB) UpdateColumns(values interface{}) *DB {
	return s.NewScope(s.Val).
		Set("aorm:update_column", true).
		Set("aorm:save_associations", false).
		InstanceSet("aorm:update_interface", values).
		callCallbacks(s.parent.callbacks.updates).db
}

// Save update value in database, if the value doesn'T have primary key, will insert it
func (s *DB) Save(value interface{}) *DB {
	scope := s.NewScope(value)
	if !scope.PrimaryKeyZero() {
		newDB := scope.callCallbacks(s.parent.callbacks.updates).db
		if newDB.Error == nil && newDB.RowsAffected == 0 {
			return s.New().FirstOrCreate(value)
		}
		return newDB
	}
	return scope.callCallbacks(s.parent.callbacks.creates).db
}

// Create insert the value into database
func (s *DB) Create(value interface{}) *DB {
	scope := s.NewScope(value)
	return scope.callCallbacks(s.parent.callbacks.creates).db
}

// Delete delete value match given conditions, if the value has primary key, then will including the primary key as condition
func (s *DB) Delete(value interface{}, where ...interface{}) *DB {
	return s.NewScope(value).
		inlineCondition(where...).
		callCallbacks(s.parent.callbacks.deletes).db
}

// Raw use raw sql as conditions, won'T run it unless invoked by other methods
//    db.Raw("SELECT name, age FROM users WHERE name = ?", 3).Scan(&result)
func (s *DB) Raw(sql string, values ...interface{}) *DB {
	return s.clone().search.Raw(true).Where(sql, values...).db
}

// Exec execute raw sql
func (s *DB) Exec(sql string, values ...interface{}) *DB {
	scope := s.NewScope(nil)
	q, err := Query{Query: sql, Args: values}.Build(scope)
	if err != nil {
		scope.Err(err)
		return scope.db
	}
	scope.Query.Query = q
	return scope.Exec().db
}

// Model specify the model you would like to run db operations
//    // update all users's name to `hello`
//    db.Model(&User{}).Update("name", "hello")
//    // if user's primary key is non-blank, will use it as condition, then will only update the user's name to `hello`
//    db.Model(&user).Update("name", "hello")
func (s *DB) Model(value interface{}) *DB {
	c := s.clone()
	c.Val = value
	return c
}

// ModelStruct specify the modelStruct and model you would like to run db operations
//    userMs := StructOf(&User{})
//    // update all users's name to `hello`
//    db.ModelStruct(userMs, &User{}).Update("name", "hello")
//    // if user's primary key is non-blank, will use it as condition, then will only update the user's name to `hello`
//    db.Model(userMs, &user).Update("name", "hello")
func (s *DB) ModelStruct(model *ModelStruct, value ...interface{}) *DB {
	c := s.clone()
	c.modelStruct = model
	c.Val = reflect.New(model.Type).Interface()
	for _, c.Val = range value {
	}
	return c
}

// Table specify the table you would like to run db operations
func (s *DB) Table(name string) *DB {
	clone := s.clone()
	clone.search.Table(name)
	clone.Val = nil
	return clone
}

// Debug start debug mode
func (s *DB) Debug() *DB {
	return s.clone().LogMode(true)
}

// Begin begin a transaction
func (s *DB) Begin() *DB {
	c := s.clone()
	if db, ok := c.db.(sqlDb); ok && db != nil {
		tx, err := db.Begin()
		c.db = interface{}(tx).(SQLCommon)

		c.dialect.SetDB(c.db)
		c.AddError(err)
	} else {
		c.AddError(ErrCantStartTransaction)
	}
	return c
}

// Commit commit a transaction
func (s *DB) Commit() *DB {
	var emptySQLTx *sql.Tx
	if db, ok := s.db.(sqlTx); ok && db != nil && db != emptySQLTx {
		if v, ok := s.values[OptKeyCommitDisabled]; ok && v.(bool) {
			s.AddError(db.Rollback())
		} else {
			s.AddError(db.Commit())
		}
	} else {
		s.AddError(ErrInvalidTransaction)
	}
	return s
}

// Transaction execute func `f` into transaction
func (s *DB) Transaction(f func(db *DB) (err error)) (err error) {
	s = s.Begin().Set("aorm:disable_scope_transaction", true)
	defer func() {
		if r := recover(); r != nil {
			s.Rollback()
			err = tracederror.New(errors.Wrap(r.(error), "transaction"))
		} else if err != nil {
			s.Rollback()
			err = errors.Wrap(err, "transaction")
		} else {
			err = s.Commit().Error
		}
	}()
	return f(s)
}

// RawTransaction execute func `f` into sql.Tx
func (s *DB) RawTransaction(f func(tx *sql.Tx) (err error)) (err error) {
	var tx *sql.Tx
	if tx, err = s.db.(*sql.DB).Begin(); err != nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			err = tracederror.New(errors.Wrap(r.(error), "transaction"))
		} else if err != nil {
			tx.Rollback()
			err = errors.Wrap(err, "transaction")
		} else {
			err = errors.Wrap(tx.Commit(), "commit")
		}
	}()
	return f(tx)
}

// Rollback rollback a transaction
func (s *DB) Rollback() *DB {
	var emptySQLTx *sql.Tx
	if db, ok := s.db.(sqlTx); ok && db != nil && db != emptySQLTx {
		s.AddError(db.Rollback())
	} else {
		s.AddError(ErrInvalidTransaction)
	}
	return s
}

// RecordNotFound check if returning ErrRecordNotFound error
func (s *DB) RecordNotFound() bool {
	if IsError(ErrRecordNotFound, s.GetErrors()...) {
		return true
	}
	return false
}

// Association start `Association Mode` to handler relations things easir in that mode, refer: https://jinzhu.github.io/gorm/associations.html#association-mode
func (s *DB) Association(relatedFieldName string) *Association {
	var err error
	var scope = s.Set("aorm:association:source", s.Val).NewScope(s.Val)
	if scope.ID().IsZero() {
		err = errors.New("primary key can'T be nil")
	} else {
		s := scope.Struct()
		if field, ok := s.FieldsByName[relatedFieldName]; ok {
			if field.Relationship == nil || len(field.Relationship.ForeignFieldNames) == 0 {
				err = fmt.Errorf("invalid association %v for %v", relatedFieldName, scope.IndirectValue().Type())
			} else {
				f := s.InstanceOf(scope.Value, field.Name).Fields[0]
				return &Association{scope: scope, column: relatedFieldName, field: f}
			}
		} else {
			err = fmt.Errorf("%v doesn'T have relatedFieldName %v", scope.IndirectValue().Type(), relatedFieldName)
		}
	}

	return (&Association{}).addErr(err)
}

// Preload preload associations with given conditions
//    db.Preload("Orders", "state NOT IN (?)", "cancelled").Find(&users)
func (s *DB) Preload(field string, options ...*InlinePreloadOptions) *DB {
	return s.clone().search.Preload(field, options...).db
}

// Preload preload associations with given conditions
//    db.Preload("Orders", "state NOT IN (?)", "cancelled").Find(&users)
func (s *DB) InlinePreload(field string, options ...*InlinePreloadOptions) *DB {
	return s.clone().search.InlinePreload(field, options...).db
}

// InlinePreloadFields set inline preload fields of value type
func (s *DB) InlinePreloadFields(value interface{}, fields ...string) *DB {
	key := InlinePreloadFieldsKeyOf(value)
	new := map[string]bool{}

	if old, ok := s.Get(key); ok {
		for k := range old.(map[string]bool) {
			new[k] = true
		}
	}

	for _, f := range fields {
		if f[0] == '-' {
			f := f[1:]
			if _, ok := new[f]; ok {
				delete(new, f)
				continue
			}
		}
		new[f] = true
	}

	return s.Set(key, new)
}

// AutoInlinePreload preload associations
func (s *DB) AutoInlinePreload(value interface{}) *DB {
	modelStruct := StructOf(value)
	var fields []string

	if data, ok := s.Get(InlinePreloadFieldsKeyOf(value)); ok {
		for k := range data.(map[string]bool) {
			fields = append(fields, k)
		}
	} else {
		if ipf, ok := value.(InlinePreloadFields); ok {
			for _, fieldName := range ipf.GetAormInlinePreloadFields() {
				if f, ok := modelStruct.FieldsByName[fieldName]; ok {
					if f.Relationship != nil {
						fields = append(fields, f.Name)
					}
				}
			}
		}

		if modelStruct.virtualFieldsAutoInlinePreload != nil {
			for _, fieldName := range modelStruct.virtualFieldsAutoInlinePreload {
				fields = append(fields, fieldName)
			}
		}
	}

	if len(fields) > 0 {
		clone := s.clone()
		for _, f := range fields {
			clone.search.InlinePreload(f)
		}
		return clone
	}

	return s
}

// Set set setting by name, which could be used in callbacks, will clone a new db, and update its setting
func (s *DB) Set(name interface{}, value interface{}) *DB {
	return s.clone().InstantSet(name, value)
}

// InstantSet instant set setting, will affect current db
func (s *DB) InstantSet(name interface{}, value interface{}) *DB {
	if s, ok := name.(string); ok {
		name = strings.ReplaceAll(s, "gorm:", "aorm:")
	}
	s.values[name] = value
	return s
}

// Get get setting by name
func (s *DB) Get(name interface{}) (value interface{}, ok bool) {
	if s, ok := name.(string); ok {
		name = strings.ReplaceAll(s, "gorm:", "aorm:")
	}
	value, ok = s.values[name]
	return
}

// GetBool get boolean setting by name ou default
func (s *DB) GetBool(name interface{}, defaul ...bool) bool {
	value, ok := s.Get(name)
	if ok {
		return value.(bool)
	}
	for _, defaul := range defaul {
		return defaul
	}
	return false
}

// MustGet mus get setting by name
func (s *DB) MustGet(name interface{}) (value interface{}) {
	if s, ok := name.(string); ok {
		name = strings.ReplaceAll(s, "gorm:", "aorm:")
	}
	return s.values[name]
}

// SetJoinTableHandler set a model's join table handler for a relation
func (s *DB) SetJoinTableHandler(source interface{}, column string, handler JoinTableHandlerInterface) {
	sourceStruct := s.StructOf(source)
	if field, ok := sourceStruct.FieldByName(column); ok {
		if field.TagSettings["M2M"] != "" {
			destination := s.StructOf(reflect.New(field.Struct.Type).Interface())
			handler.Setup(field.Relationship, DefaultM2MNamer(field), sourceStruct, destination)
			field.Relationship.JoinTableHandler = handler
			if table := handler.Table(s); s.Dialect().HasTable(table) {
				s.Table(table).AutoMigrate(handler)
			}
		}
	}
}

// AddError add error to the db
func (s *DB) AddError(err error) error {
	if err != nil {
		if !IsError(ErrRecordNotFound, err) {
			if s.logMode == 0 {
				s.print(fileWithLineNum() + ": " + err.Error())
			} else {
				s.log(err)
			}

			errors := Errors(s.GetErrors()).Add(err)
			if len(errors) > 1 {
				err = errors
			}
		}

		s.Error = err
	}
	return err
}

// GetErrors get happened errors from the db
func (s *DB) GetErrors() []error {
	if errs, ok := s.Error.(Errors); ok {
		return errs
	} else if s.Error != nil {
		return []error{s.Error}
	}
	return []error{}
}

// ScopeErrorCallback register error callback for scope
func (s *DB) ScopeErrorCallback(cb func(scope *Scope, err error)) *DB {
	s = s.clone()
	key := "aorm:scope_error_callbacks"
	var callbacks []func(scope *Scope, err error)
	if v, ok := s.Get(key); ok {
		callbacks = append(callbacks, v.([]func(scope *Scope, err error))...)
	}
	callbacks = append(callbacks, cb)
	s.Set(key, callbacks)
	return s
}

////////////////////////////////////////////////////////////////////////////////
// Private Methods For DB
////////////////////////////////////////////////////////////////////////////////

func (s DB) clone() *DB {
	values := map[interface{}]interface{}{}
	for key, value := range s.values {
		values[key] = value
	}
	s.values = values

	if s.search == nil {
		s.search = &search{limit: -1, offset: -1}
	} else {
		s.search = s.search.clone()
	}

	s.search.db = &s
	return &s
}

func (s *DB) print(v ...interface{}) {
	s.logger.Debug(v...)
}

func (s *DB) log(v ...interface{}) {
	if s != nil && s.logMode == 2 {
		s.logger.Info(append([]interface{}{fileWithLineNum()}, v...)...)
	}
}

func (s *DB) slog(sql string, t time.Time, vars ...interface{}) {
	if s.logMode == 2 {
		s.print("SQL", fileWithLineNum(), NowFunc().Sub(t), sql, vars, s.RowsAffected)
	}
}

// Disable after scan callback. If typs not is empty, disable for typs, other else, disable for all
func (s *DB) DisableAfterScanCallback(typs ...interface{}) *DB {
	key := "aorm:disable_after_scan"

	s = s.clone()

	if len(typs) == 0 {
		s.values[key] = true
		return s
	}

	for _, typ := range typs {
		rType := indirectType(reflect.TypeOf(typ))
		s.values[key+":"+rType.PkgPath()+"."+rType.Name()] = true
	}

	return s
}

// Enable after scan callback. If typs not is empty, enable for typs, other else, enable for all.
// The disabled types will not be enabled unless they are specifically informed.
func (s *DB) EnableAfterScanCallback(typs ...interface{}) *DB {
	key := "aorm:disable_after_scan"

	s = s.clone()

	if len(typs) == 0 {
		s.values[key] = false
		return s
	}

	for _, typ := range typs {
		rType := indirectType(reflect.TypeOf(typ))
		s.values[key+":"+rType.PkgPath()+"."+rType.Name()] = false
	}

	return s
}

// Return if after scan callbacks has be enable. If typs is empty, return default, other else, return for informed
// typs.
func (s *DB) IsEnabledAfterScanCallback(typs ...interface{}) (ok bool) {
	key := "aorm:disable_after_scan"

	if v, ok := s.values[key]; ok {
		return !v.(bool)
	}

	for _, typ := range typs {
		rType := indirectType(reflect.TypeOf(typ))
		v, ok := s.values[key+":"+rType.PkgPath()+"."+rType.Name()]
		if ok && v.(bool) {
			return false
		}
	}

	return true
}

func (s *DB) Path() []interface{} {
	return s.path
}

func (s *DB) PathString() string {
	ps := make([]string, len(s.path))
	for i, p := range s.path {
		if pl, ok := p.(interface{ GetLabel() string }); ok {
			ps[i] = pl.GetLabel()
		} else if pl, ok := p.(interface{ GetPrivateLabel() string }); ok {
			ps[i] = pl.GetPrivateLabel()
		} else if pl, ok := p.(interface{ GetName() string }); ok {
			ps[i] = pl.GetName()
		} else {
			ps[i] = fmt.Sprint(p)
		}
	}
	return strings.Join(ps, " > ")
}

func (s *DB) Inside(name ...interface{}) *DB {
	clone := s.clone()
	clone.path = append(clone.path, name...)
	return clone
}

func (s *DB) RegisterAssigner(assigners ...Assigner) *DB {
	s = s.clone()
	if s.assigners == nil {
		s.assigners = &AssignerRegistrator{}
	} else {
		s.assigners = &AssignerRegistrator{s.assigners.data}
	}
	s.assigners.Register(assigners...)
	return s
}

func (s *DB) GetAssigner(typ reflect.Type) (assigner Assigner) {
	if s.assigners != nil {
		if assigner = s.assigners.Get(typ); assigner != nil {
			return
		}
	}
	if assigner = s.dialect.GetAssigner(typ); assigner != nil {
		return
	}
	return assigners.Get(typ)
}

func (s *DB) GetArgVariabler(arg interface{}) (argVar ArgBinder) {
	var ok bool
	if argVar, ok = arg.(ArgBinder); ok {
		return
	} else if assigner := s.GetAssigner(reflect.TypeOf(arg)); assigner != nil {
		if argVar, ok = assigner.(ArgBinder); ok {
			return
		}
	}
	return
}

func (s *DB) SingleUpdate() bool {
	if v, ok := s.values[OptKeySingleUpdate]; !ok || v.(bool) {
		return true
	}
	return false
}

func (s *DB) Loggers(tableName string, set ...bool) (clone *DB, sl *ScopeLoggers, ok bool) {
	key := scopeLoggerKey(tableName) + ":global"
	if cbsv, ok := s.Get(key); ok {
		return s, cbsv.(*ScopeLoggers), true
	} else if len(set) > 0 && set[0] {
		sl = &ScopeLoggers{}
		return s.Set(key, sl), sl, false
	}
	return s, sl, false
}

func (s *DB) SetCurrentUser(user ID) *DB {
	return s.Set("aorm:current_user", user)
}

func (s *DB) GetCurrentUser() (user ID, ok bool) {
	if v, ok := s.Get("aorm:current_user"); ok {
		user = v.(ID)
		return user, user != nil
	}
	return
}

func (s *DB) DefaultColumnValue(valuer func(scope *Scope, record interface{}, column string) interface{}) *DB {
	s = s.clone()
	s.search.defaultColumnValue = valuer
	return s
}

func (s *DB) ColumnsScannerCallback(cb func(scope *Scope, record interface{}, columns []string, values []interface{})) *DB {
	s = s.clone()
	s.search.columnsScannerCallback = cb
	return s
}

// Opt apply options to db
func (s *DB) Opt(opt ...Opt) *DB {
	for _, opt := range opt {
		s = opt.Apply(s)
	}
	return s
}
