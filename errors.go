package aorm

import (
	"errors"
	"fmt"
	"reflect"

	error_utils "github.com/unapu-go/error-utils"
)

var (
	// ErrRecordNotFound record not found error, happens when haven'T find any matched data when looking up with a struct
	ErrRecordNotFound = errors.New("record not found")
	// ErrInvalidSQL invalid SQL error, happens when you passed invalid SQL
	ErrInvalidSQL = errors.New("invalid SQL")
	// ErrInvalidTransaction invalid transaction when you are trying to `Commit` or `Rollback`
	ErrInvalidTransaction = errors.New("no valid transaction")
	// ErrCantStartTransaction can'T start transaction when you are trying to start one with `Begin`
	ErrCantStartTransaction = errors.New("can'T start transaction")
	// ErrUnaddressable unaddressable value
	ErrUnaddressable = errors.New("using unaddressable value")
	// ErrSingleUpdateKey single UPDATE require primary key value
	ErrSingleUpdateKey = errors.New("Single UPDATE require primary key value.")

	IsError     = error_utils.IsError
	ErrorByType = error_utils.ErrorByType
)

// Errors contains all happened errors
type Errors = error_utils.Errors

// IsRecordNotFoundError returns current error has record not found error or not
func IsRecordNotFoundError(err error) bool {
	return IsError(ErrRecordNotFound, err)
}

// Represents Query error
type QueryError struct {
	QueryInfo
	cause error
}

func NewQueryError(cause error, q Query, varBinder func(i int) string) *QueryError {
	qi := NewQueryInfo(q, varBinder)
	return &QueryError{*qi, cause}
}

// Returns the original error
func (e *QueryError) Cause() error {
	return e.cause
}

func (e *QueryError) Error() string {
	return e.cause.Error() + "\n" + (Query{e.Query.Query, NamedStringerArgs(&e.QueryInfo)}.String())
}

func IsQueryError(err ...error) bool {
	return ErrorByType(reflect.TypeOf(QueryError{}), err...) != nil
}

func GetQueryError(err ...error) *QueryError {
	if result := ErrorByType(reflect.TypeOf(QueryError{}), err...); result != nil {
		return result.(*QueryError)
	}
	return nil
}

type DuplicateUniqueIndexError struct {
	index *StructIndex
	cause error
}

func (d DuplicateUniqueIndexError) Index() *StructIndex {
	return d.index
}

func (d DuplicateUniqueIndexError) Cause() error {
	return d.cause
}

func (d DuplicateUniqueIndexError) Error() string {
	return "duplicate unique index of " + d.index.Model.Type.PkgPath() +
		"." + d.index.Model.Type.Name() + " " + fmt.Sprint(d.index.FieldsNames()) +
		" caused by: " + d.cause.Error()
}

func IsDuplicateUniqueIndexError(err ...error) bool {
	return ErrorByType(reflect.TypeOf(DuplicateUniqueIndexError{}), err...) != nil
}

func GetDuplicateUniqueIndexError(err ...error) *DuplicateUniqueIndexError {
	if result := ErrorByType(reflect.TypeOf(DuplicateUniqueIndexError{}), err...); result != nil {
		return result.(*DuplicateUniqueIndexError)
	}
	return nil
}

type PathError interface {
	error
	Path() string
}
