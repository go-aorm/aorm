package aorm

import (
	"fmt"
	"strings"
)

type ForeignKeyDefinition struct {
	Name               string
	SrcTableName       string
	SrcColumns         []string
	DstTableName       string
	DstColumns         []string
	OnUpdate, OnDelete string
}

func (this *ForeignKeyDefinition) Query(d Dialector) string {
	var fkFields_ = make([]string, len(this.SrcColumns))
	for i, name := range this.SrcColumns {
		fkFields_[i] = QuoteIfPossible(d, name)
	}
	var destFields_ = make([]string, len(this.DstColumns))
	for i, name := range this.DstColumns {
		destFields_[i] = QuoteIfPossible(d, name)
	}
	return fmt.Sprintf(`ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s) ON DELETE %s ON UPDATE %s;`,
		QuoteIfPossible(d, this.SrcTableName),
		QuoteIfPossible(d, this.Name),
		strings.Join(fkFields_, ","),
		QuoteIfPossible(d, this.DstTableName),
		strings.Join(destFields_, ","),
		this.OnDelete,
		this.OnUpdate)
}

func (this *ForeignKeyDefinition) Create(db *DB) error {
	if db.Dialect().HasForeignKey(this.SrcTableName, this.Name) {
		return nil
	}
	return db.Exec(this.Query(db.dialect)).Error
}

type ForeignKey struct {
	Name               string
	Prepare            func(srcScope, dstScope *Scope, def *ForeignKeyDefinition)
	Field              *StructField
	OnUpdate, OnDelete string
}

func (this *ForeignKey) Definition(scope *Scope) (def ForeignKeyDefinition) {
	rel := this.Field.Relationship
	switch rel.Kind {
	case "has_many":
		dstScope := scope.db.NewModelScope(rel.AssociationModel, rel.AssociationModel.Value)
		srcScope := scope.db.NewModelScope(rel.Model, rel.Model.Value)
		def.SrcTableName = srcScope.TableName()
		def.DstTableName = dstScope.TableName()
		def.SrcColumns = rel.ForeignDBNames
		def.DstColumns = rel.AssociationForeignDBNames
		def.OnDelete = this.OnDelete
		def.OnUpdate = this.OnUpdate
		if def.Name = this.Name; def.Name == "" {
			if this.Prepare != nil {
				this.Prepare(srcScope, dstScope, &def)
			}
			if def.Name == "" {
				def.Name = ForeignKeyNameOf(def.SrcTableName, def.DstTableName)
			}
		}
	case "has_one":
		dstScope := scope.db.NewModelScope(rel.Model, rel.Model.Value)
		srcScope := scope.db.NewModelScope(rel.AssociationModel, rel.AssociationModel.Value)
		def.SrcTableName = srcScope.TableName()
		def.DstTableName = dstScope.TableName()
		def.SrcColumns = rel.ForeignDBNames
		def.DstColumns = rel.AssociationForeignDBNames
		def.OnDelete = this.OnDelete
		def.OnUpdate = this.OnUpdate
		if def.Name = this.Name; def.Name == "" {
			if this.Prepare != nil {
				this.Prepare(srcScope, dstScope, &def)
			}
			if def.Name == "" {
				def.Name = ForeignKeyNameOf(def.SrcTableName, def.DstTableName)
			}
		}
	case "belongs_to":
		dstScope := scope.db.NewModelScope(rel.AssociationModel, rel.AssociationModel.Value)
		srcScope := scope.db.NewModelScope(rel.Model, rel.Model.Value)
		def.SrcTableName = srcScope.TableName()
		def.DstTableName = dstScope.TableName()
		def.SrcColumns = rel.ForeignDBNames
		def.DstColumns = rel.AssociationForeignDBNames
		def.OnDelete = this.OnDelete
		def.OnUpdate = this.OnUpdate
		if def.Name = this.Name; def.Name == "" {
			if this.Prepare != nil {
				this.Prepare(srcScope, dstScope, &def)
			}
			if def.Name == "" {
				def.Name = ForeignKeyNameOf(def.SrcTableName, def.DstTableName)
			}
		}
	}
	return
}

func ForeignKeyNameOf(srcTable, dstTable string) (name string) {
	_, name = JoinNameOfString(srcTable, dstTable)
	return "fkc_" + name
}
