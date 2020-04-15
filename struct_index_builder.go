package aorm

import (
	"fmt"
	"strings"
)

type sutructIndexesBuilder struct {
	unique   bool
	tagAlias []string
	tagName  string
	indexes  map[string]*struct {
		name          string
		fields        []string
		where         []string
		whereTemplate []string
	}
}

func newSutructIndexesBuilder(unique bool, tagName string, tagAlias ...string) *sutructIndexesBuilder {
	return &sutructIndexesBuilder{
		unique:   unique,
		tagName:  tagName,
		tagAlias: tagAlias,
		indexes: map[string]*struct {
			name          string
			fields        []string
			where         []string
			whereTemplate []string
		}{},
	}
}

func (this *sutructIndexesBuilder) readField(field *StructField) {
	for _, tag := range this.tagAlias {
		if value, ok := field.TagSettings[tag]; ok {
			if value == tag {
				value = this.tagName
			}
			field.TagSettings[this.tagName] = value
			delete(field.TagSettings, tag)
		}
	}
	if name, ok := field.TagSettings[this.tagName]; ok {
		names := strings.Split(name, ",")

		for _, name := range names {
			ix := &struct {
				name          string
				fields        []string
				where         []string
				whereTemplate []string
			}{}
			parts := strings.SplitN(name, "=", 2)
			if len(parts) == 2 {
				name = parts[0]
				ix.where = append(ix.where, strings.ReplaceAll(parts[1], "{}", QuoteCharS+field.DBName+QuoteCharS))
				ix.whereTemplate = append(ix.whereTemplate, strings.ReplaceAll(parts[1], "{}", field.Name))
			}
			ix.fields = append(ix.fields, field.DBName)
			if name != "" && name != this.tagName {
				// named
				ix.name = name
				if old, ok := this.indexes[ix.name]; ok {
					old.fields = append(old.fields, ix.fields...)
					old.where = append(old.where, ix.where...)
				} else {
					this.indexes[ix.name] = ix
				}
			} else {
				// unnamed
				this.indexes["!"+field.Name] = ix
			}
		}
	}
}

func (this *sutructIndexesBuilder) build(modelStruct *ModelStruct) (indexes IndexMap, err error) {
	indexes = make(IndexMap)

	for key, schema := range this.indexes {
		ix := &StructIndex{
			Model:         modelStruct,
			Unique:        this.unique,
			Where:         strings.Join(schema.where, " AND "),
			WhereTemplate: strings.Join(schema.whereTemplate, " AND "),
		}

		if key[0] != '!' {
			ix.NameTemplate = key
		}

		for _, fieldName := range schema.fields {
			if field, ok := modelStruct.FieldByName(fieldName); ok {
				ix.Fields = append(ix.Fields, field)
			} else {
				return nil, fmt.Errorf("build index %q for %s.%s failed: field or column %q does not exists",
					key, modelStruct.Type.PkgPath(), modelStruct.Type.Name(), fieldName)
			}
		}

		indexes[key] = ix
	}

	return
}
