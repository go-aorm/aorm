package aorm_test

import (
	"fmt"
	"github.com/moisespsena-go/aorm"
	"testing"
)

func TestPathsFromQuery(t *testing.T) {
	paths := []string{"F1", "F2.F3", "F4.F5.F6"}
	result := aorm.PathsFromQuery("{F1} = 1 and {F2.F3} and {F2.F3} and {F4.F5.F6}")
	if len(paths) != len(result) {
		t.Fail()
	}
	for i, p := range paths {
		if p != result[i] {
			t.Fail()
			return
		}
	}
}

func TestWithInlineQuery_Merge(t *testing.T) {
	ip := &aorm.InlinePreloads{}
	ip.Next("F1")
	ip.Next("F2", "F3")
	ip.Next("F4", "F5", "F6")
	iq := aorm.IQ("{F1}.status = 1 and {F2.F3}.id !=  and {F2.F3} and {F4.F5.F6}.ok")
	query := iq.Merge(ip)
	fmt.Println(query)
}

func TestInlinePreloader_GetQuery(t *testing.T) {
	db := DB
	type A struct {
		aorm.Model
		V1 int
	}
	type B struct {
		aorm.Model
		V2 int
	}
	db = db.Model(&A{})
	db = db.InlinePreload("B")
	scope := db.NewScope(&A{})
	scope.SetVirtualField("B", &B{})
	aorm.InlinePreloadCallback(scope)
	println()
}