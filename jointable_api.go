package aorm

// JoinTableHandlerInterface is an interface for how to handle many2many relations
type JoinTableHandlerInterface interface {
	TableNamer

	// initialize join table handler
	Setup(relationship *Relationship, tableName string, source, destination *ModelStruct)
	// Table return join table's table name
	Table(db *DB) string
	// Add create relationship in join table for source and destination
	Add(handler JoinTableHandlerInterface, db *DB, source interface{}, destination interface{}) error
	// Delete delete relationship in join table for sources
	Delete(handler JoinTableHandlerInterface, db *DB, sources ...interface{}) error
	// JoinWith query with `Join` conditions
	JoinWith(handler JoinTableHandlerInterface, db *DB, source interface{}) *DB
	// SourceForeignKeys return source foreign keys
	SourceForeignKeys() []JoinTableForeignKey
	// DestinationForeignKeys return destination foreign keys
	DestinationForeignKeys() []JoinTableForeignKey

	Source() JoinTableSource
	Destination() JoinTableSource
}
