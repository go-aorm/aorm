package aorm

const tableNameResolvers string = "aorm:table_name_resolvers"

type TableNameResolverFunc func(singular bool, value interface{}) (tableName string, ok bool)

func (this TableNameResolverFunc) TableName(singular bool, value interface{}) (tableName string, ok bool) {
	return this(singular, value)
}

func TableNameResolverOf(f func(singular bool, value interface{}) (tableName string, ok bool)) TableNameResolver {
	return TableNameResolverFunc(f)
}

func (db *DB) SetTableNameResolver(resolver TableNameResolver) *DB {
	if old, ok := db.Get(tableNameResolvers); ok {
		return db.Set(tableNameResolvers, append(old.([]TableNameResolver), resolver))
	} else {
		return db.Set(tableNameResolvers, []TableNameResolver{resolver})
	}
}
