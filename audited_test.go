package aorm_test

import (
	"fmt"
	"testing"

	"github.com/moisespsena-go/aorm"
)

type AuditedProduct struct {
	aorm.SoftDeleteAuditedModel
	Name string
}

type SimpleUser struct {
	aorm.Model
	Name string
}

func TestCreateUser(t *testing.T) {
	db := DB
	db.DropTable(&SimpleUser{}, &AuditedProduct{})
	db.AutoMigrate(&SimpleUser{}, &AuditedProduct{})

	user := SimpleUser{Name: "user1"}
	db.Save(&user)
	db = db.Set("gorm:current_user", user)

	product := AuditedProduct{Name: "product1"}
	db.Save(&product)
	if product.CreatedBy != fmt.Sprintf("%v", user.ID) {
		t.Errorf("created_by is not equal current user")
	}

	product.Name = "product_new"
	db.Save(&product)
	if *product.UpdatedBy != fmt.Sprintf("%v", user.ID) {
		t.Errorf("updated_by is not equal current user")
	}

	db.Delete(&product)
	if *product.DeletedBy != fmt.Sprintf("%v", user.ID) {
		t.Errorf("deleted_by is not equal current user")
	}
}
