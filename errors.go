package aorm

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var (
	// ErrRecordNotFound record not found error, happens when haven't find any matched data when looking up with a struct
	ErrRecordNotFound = errors.New("record not found")
	// ErrInvalidSQL invalid SQL error, happens when you passed invalid SQL
	ErrInvalidSQL = errors.New("invalid SQL")
	// ErrInvalidTransaction invalid transaction when you are trying to `Commit` or `Rollback`
	ErrInvalidTransaction = errors.New("no valid transaction")
	// ErrCantStartTransaction can't start transaction when you are trying to start one with `Begin`
	ErrCantStartTransaction = errors.New("can't start transaction")
	// ErrUnaddressable unaddressable value
	ErrUnaddressable = errors.New("using unaddressable value")
)

// Errors contains all happened errors
type Errors []error

func WalkErr(cb func(err error) (stop bool), errs ...error) (stop bool) {
	for _, err := range errs {
		if err == nil {
			continue
		}

		if cb(err) {
			return true
		}

		if err, ok := err.(interface{ Err() error }); ok {
			if WalkErr(cb, err.Err()) {
				return true
			}
		}

		if errs, ok := err.(Errors); ok {
			if WalkErr(cb, errs...) {
				return true
			}
		} else if errs, ok := err.(interface{ Errors() []error }); ok {
			if WalkErr(cb, errs.Errors()...) {
				return true
			}
		} else if errs, ok := err.(interface{ GetErrors() []error }); ok {
			if WalkErr(cb, errs.GetErrors()...) {
				return true
			}
		}
	}
	return false
}

func IsError(expected error, err ...error) (is bool) {
	return WalkErr(func(err error) (stop bool) {
		return err == expected
	}, err...)
}

// IsRecordNotFoundError returns current error has record not found error or not
func IsRecordNotFoundError(err error) bool {
	return IsError(ErrRecordNotFound, err)
}

// GetErrors gets all happened errors
func (errs Errors) GetErrors() []error {
	return errs
}

// Add adds an error
func (errs Errors) Add(newErrors ...error) Errors {
	for _, err := range newErrors {
		if err == nil {
			continue
		}

		if errors, ok := err.(Errors); ok {
			errs = errs.Add(errors...)
		} else {
			ok = true
			for _, e := range errs {
				if err == e {
					ok = false
				}
			}
			if ok {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

// Error format happened errors
func (errs Errors) Error() string {
	var errors = []string{}
	for _, e := range errs {
		errors = append(errors, e.Error())
	}
	return strings.Join(errors, "; ")
}

// Represents Query Error
type QueryError struct {
	err  error
	SQL  string
	Args []interface{}
}

// Returns the original error
func (e QueryError) Err() error {
	return e.err
}

func (e QueryError) Error() string {
	var b bytes.Buffer
	b.WriteString(e.err.Error() + "\nBEGIN SQL >>\n" + e.SQL + "\n<< END SQL")
	if len(e.Args) > 0 {
		b.WriteString("\nSQL Args:\n")
		for i, arg := range e.Args {
			typ := reflect.TypeOf(arg)
			b.WriteString(fmt.Sprintf("  - %v: {%v[%s]}\n", i, indirectType(typ).PkgPath(), typ))
			argValue := strings.Split(fmt.Sprint(arg), "\n")[0]
			if len(argValue) > 50 {
				argValue = argValue[0:50] + " ..."
			}
			b.WriteString("    " + argValue)
		}
	}
	return b.String()
}
