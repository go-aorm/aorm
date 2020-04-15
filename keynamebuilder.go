package aorm

import (
	"fmt"
	"regexp"
	"strings"
)

type (
	KeyNamer interface {
		// BuildKeyName returns a valid key name (foreign key, index key) for the given table, field and reference
		BuildKeyName(kind, tableName string, fields ...string) string
	}

	KeyNamerFunc = func(kind, tableName string, fields ...string) string
	keyNamerFunc KeyNamerFunc

	// DefaultKeyNamer contains the default foreign key name generator method
	DefaultKeyNamer struct{}
)

var keyNameRegex = regexp.MustCompile("[^a-zA-Z0-9]+")

func (this keyNamerFunc) BuildKeyName(kind, tableName string, fields ...string) string {
	return this(kind, tableName, fields...)
}

func KeyNameBuilderOf(f KeyNamerFunc) KeyNamer {
	return keyNamerFunc(f)
}

// BuildKeyName returns a valid key name (foreign key, index key) for the given table, field and reference
func (DefaultKeyNamer) BuildKeyName(kind, tableName string, fields ...string) string {
	keyName := fmt.Sprintf("%s_%s_%s", kind, tableName, strings.Join(fields, "_"))
	keyName = keyNameRegex.ReplaceAllString(keyName, "_")
	return keyName
}
