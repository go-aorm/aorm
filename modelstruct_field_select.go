package aorm

func (this *StructField) Select(scope *Scope, tableName string) Query {
	if this.Selector != nil {
		if this.IsReadOnly {
			return this.Selector.Select(this, scope, tableName).Wrap("(", ") AS "+this.DBName)
		}
		return this.Selector.Select(this, scope, tableName)
	}
	return Query{Query: tableName + this.DBName}
}

func (this *StructField) SelectWrap(scope *Scope, query *Query) *Query {
	if this.SelectWraper != nil {
		return this.SelectWraper.SelectWrap(this, scope, query)
	}
	return query
}
