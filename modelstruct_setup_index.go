package aorm

import (
	"github.com/pkg/errors"
)

func (this *ModelStruct) setupIndexes() (err error) {
	var (
		indexes       = newSutructIndexesBuilder(false, "INDEX", "IX")
		uniqueIndexes = newSutructIndexesBuilder(true, "UNIQUE_INDEX", "UNIQUE", "UX")
	)

	for _, field := range this.Fields {
		indexes.readField(field)
		uniqueIndexes.readField(field)
	}

	if this.Indexes, err = indexes.build(this); err != nil {
		return errors.Wrap(err, "INDEX")
	}
	if this.UniqueIndexes, err = uniqueIndexes.build(this); err != nil {
		return errors.Wrap(err, "UNIQUE INDEX")
	}

	return
}
