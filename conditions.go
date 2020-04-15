package aorm

import (
	"strings"
)

type Conditions struct {
	whereConditions []*Clause
	orConditions    []*Clause
	notConditions   []*Clause
	Args            Vars
}

func NewConditions() *Conditions {
	return &Conditions{}
}

func (this *Conditions) Prepare(cb func(c *Conditions, query interface{}, args []interface{}, replace func(query interface{}, args ...interface{}))) {
	iter := func(c *Conditions, l *[]*Clause) {
		for i, cond := range *l {
			cb(c, cond.Query, cond.Args, func(query interface{}, args ...interface{}) {
				if query == nil {
					(*l)[i] = nil
				} else {
					(*l)[i].Query, (*l)[i].Args = query, args
				}
			})
		}
	}

	var (
		where, or, not []*Clause
		args           []interface{}
	)
	cur := this
	for cur.hasConditions() {
		newC := &Conditions{}
		iter(newC, &cur.whereConditions)
		iter(newC, &cur.orConditions)
		iter(newC, &cur.notConditions)
		where, or, not = append(where, cur.whereConditions...), append(or, cur.orConditions...), append(not, cur.notConditions...)
		args = append(args, cur.Args...)
		cur = newC
	}
	this.whereConditions, this.orConditions, this.notConditions = where, or, not
}

func (this *Conditions) Where(query interface{}, values ...interface{}) *Conditions {
	this.whereConditions = append(this.whereConditions, &Clause{query, values})
	return this
}

func (this *Conditions) Not(query interface{}, values ...interface{}) *Conditions {
	this.notConditions = append(this.notConditions, &Clause{query, values})
	return this
}

func (this *Conditions) Or(query interface{}, values ...interface{}) *Conditions {
	this.orConditions = append(this.orConditions, &Clause{query, values})
	return this
}

func (this *Conditions) hasConditions() bool {
	return len(this.whereConditions) > 0 ||
		len(this.orConditions) > 0 ||
		len(this.notConditions) > 0
}

func (this *Conditions) MergeTo(db *DB) *DB {
	db = db.clone()
	db.search.whereConditions = append(db.search.whereConditions, this.whereConditions...)
	db.search.orConditions = append(db.search.orConditions, this.orConditions...)
	db.search.notConditions = append(db.search.notConditions, this.notConditions...)
	return db
}

func (this *Conditions) WhereClause(scope *Scope) (result Query) {
	var (
		andConditions, orConditions []string
	)

	for _, clause := range this.whereConditions {
		q := clause.BuildCondition(scope, true)
		andConditions = append(andConditions, q.Query)
		result.AddArgs(q.Args...)
	}

	for _, clause := range this.orConditions {
		q := clause.BuildCondition(scope, true)
		orConditions = append(orConditions, q.Query)
		result.AddArgs(q.Args...)
	}

	for _, clause := range this.notConditions {
		q := clause.BuildCondition(scope, false)
		andConditions = append(andConditions, q.Query)
		result.AddArgs(q.Args...)
	}

	orSQL := strings.Join(orConditions, " OR ")
	combinedSQL := strings.Join(andConditions, " AND ")
	if len(combinedSQL) > 0 {
		if len(orSQL) > 0 {
			combinedSQL = combinedSQL + " OR " + orSQL
		}
	} else {
		combinedSQL = orSQL
	}
	result.Query = combinedSQL
	return
}
