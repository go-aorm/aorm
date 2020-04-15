package aorm_test

import (
	"fmt"
)

func init() {

}

func ExampleStructFieldMethodCallbacksRegistrator_DisableFieldType() {
	fmt.Println(`if registrator.IsEnabledFieldType(&Media{}) {
	registrator.DisableFieldType(&Media{})
}`)
}

func ExampleStructFieldMethodCallbacksRegistrator_EnabledFieldType() {
	fmt.Println(`if !registrator.IsEnabledFieldType(&Media{}) {
	println("not enabled")
}`)
}

func ExampleStructFieldMethodCallbacksRegistrator_EnableFieldType() {
	fmt.Println(`if !registrator.IsEnabledFieldType(&Media{}) {
	registrator.EnableFieldType(&Media{})
}`)
}

func ExampleStructFieldMethodCallbacksRegistrator_RegisteredFieldType() {
	fmt.Println(`
if registrator.RegisteredFieldType(&Media{}) {
	println("not registered")
}`)
}

func ExampleStructFieldMethodCallbacksRegistrator_RegisterFieldType() {
	fmt.Println("registrator.RegisterFieldType(&Media{})")
}

func ExampleAfterScanMethodCallback() {
	println(`
package main

import (
	"reflect"
	"github.com/moisespsena-go/aorm"
	"database/sql/driver"
	_ "github.com/moisespsena-go/aorm/dialects/sqlite"
	"strconv"
	"strings"
)

type Media struct {
	Name      string
	baseUrl   *string
	modelType reflect.Type
	model interface {
		GetID() int
	}
	fieldName *string
}

func (image *Media) Scan(value interface{}) error {
	image.Name = string(value.([]byte))
	return nil
}

func (image *Media) Value() (driver.Value, error) {
	return image.Name, nil
}

func (image *Media) AfterScan(scope *aorm.Scope, field *aorm.Field) {
	image.fieldName, image.model = &field.StructField.Name, scope.Val.(interface {
		GetID() int
	})
	baseUrl, _ := scope.DB().Get("base_url")
	image.baseUrl = baseUrl.(*string)
	image.modelType = reflect.TypeOf(scope.Val)
	for image.modelType.Kind() == reflect.Ptr {
		image.modelType = image.modelType.Elem()
	}
}

func (image *Media) URL() string {
	return strings.Join([]string{*image.baseUrl, image.modelType.Name(), strconv.Itoa(image.model.GetID()),
		*image.fieldName, image.Name}, "/")
}

type User struct {
	BID        int
	MainImage Media
}

func (user *User) GetID() int {
	return user.BID
}

func main() {
	// register media type
	aorm.StructFieldMethodCallbacks.RegisterFieldType(&Media{})

	db, err := aorm.Open("sqlite3", "db.db")
	if err != nil {
		panic(err)
	}

	db.AutoMigrate(&User{})

	baseUrl := "http://example.com/media"
	db = db.Set("base_url", &baseUrl)

	var model User
	db_ := db.Where("id = ?", 1).First(&model)
	if db_.RecordNotFound() {
		db.Save(&User{MainImage: Media{Name: "picture.jpg"}})
		err = db.Where("id = ?", 1).First(&model).error
		if err != nil {
			panic(err)
		}
	} else if db_.error != nil {
		panic(db_.error)
	}

	println("Media URL:", model.MainImage.URL())
}
`)
}
