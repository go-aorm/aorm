package aorm

import (
	"fmt"
	"strings"
)

type StructIndex struct {
	Model         *ModelStruct
	Unique        bool
	NameTemplate  string
	Fields        []*StructField
	WhereTemplate string
	Where         string
}

func (this *StructIndex) BuildName(d KeyNamer, tableName string) (name string) {
	if this.NameTemplate == "" {
		var prefix string
		if this.Unique {
			prefix = "u"
		}
		return d.BuildKeyName(prefix+"ix", tableName, this.Columns()...)
	} else {
		return strings.ReplaceAll(name, "TB", tableName)
	}
}

func (this *StructIndex) Columns() []string {
	columns := make([]string, len(this.Fields))
	for i, f := range this.Fields {
		columns[i] = f.DBName
	}
	return columns
}

func (this *StructIndex) FieldsNames() []string {
	names := make([]string, len(this.Fields))
	for i, f := range this.Fields {
		names[i] = f.Name
	}
	return names
}

func (this *StructIndex) String() (s string) {
	if this.Unique {
		s = "UIX "
	} else {
		s = "IX "
	}
	if this.NameTemplate != "" {
		s += `"` + this.NameTemplate + `" `
	}
	if len(this.Fields) == 1 {
		s += this.Fields[0].Name + " "
	} else {
		fields := fmt.Sprint(this.FieldsNames())[1:]
		s += "(" + fields[0:len(fields)-1] + ") "
	}
	if this.Where != "" {
		s += "= (" + this.WhereTemplate + ")"
	}
	return strings.TrimSpace(s)
}

func (this *StructIndex) SqlCreate(d Dialector, tableName string) (sql string) {
	sql = "CREATE "
	if this.Unique {
		sql += "UNIQUE "
	}
	sql += "INDEX " + Quote(d, this.BuildName(d, tableName)) + " ON " + Quote(d, tableName) + "("
	columns := make([]string, len(this.Fields))
	for i, f := range this.Fields {
		columns[i] = Quote(d, f.DBName)
	}
	sql += strings.Join(columns, ", ") + ") "
	sql += this.Where
	return sql
}

func (this *StructIndex) SqlDrop(d interface {
	Quoter
	KeyNamer
}, tableName string) (sql string) {
	return "DROP INDEX " + Quote(d, this.BuildName(d, tableName))
}

type IndexMap map[string]*StructIndex

// FromDbName return index from db name
func (this IndexMap) FromDbName(namer KeyNamer, tableName, indexDbName string) (ix *StructIndex) {
	for _, ix := range this {
		if ix.BuildName(namer, tableName) == indexDbName {
			return ix
		}
	}
	return
}

// FromColumns return index from db column names
func (this IndexMap) FromColumns(columns ...string) (ix *StructIndex) {
main:
	for _, ix := range this {
		if len(ix.Fields) == len(columns) {
			for _, f := range ix.Fields {
				var matched bool
				for _, c := range columns {
					if c == f.DBName {
						matched = true
						break
					}
				}
				if !matched {
					continue main
				}
			}
			return ix
		}
	}
	return
}
