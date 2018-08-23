package aorm_test

import (
	"errors"
	"testing"

	"github.com/moisespsena-go/aorm"
)

func TestErrorsCanBeUsedOutsideGorm(t *testing.T) {
	errs := []error{errors.New("First"), errors.New("Second")}

	gErrs := aorm.Errors(errs)
	gErrs = gErrs.Add(errors.New("Third"))
	gErrs = gErrs.Add(gErrs)

	if gErrs.Error() != "First; Second; Third" {
		t.Fatalf("Gave wrong error, got %s", gErrs.Error())
	}
}
