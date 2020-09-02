package aorm

import (
	"fmt"
	"reflect"

	"github.com/moisespsena-go/bid"
)

type UserLocale struct {
	Model
	UserID bid.BID
	User   *User
}

type User struct {
	Model
	Locales []*UserLocale
}

func ExampleRelationship_InstanceToRelatedID() {
	var user = &User{}
	user.ID.ParseString("XqjMd3wdyUIWWM8N")
	var locale = &UserLocale{}
	locale.ID.ParseString("XqjM4HwdyUY6z-AD")
	fmt.Println(user.ID, locale.ID)

	model := StructOf(user)
	instance := model.InstanceOf(user)
	rel := model.FieldsByName["Locales"].Relationship
	ID := rel.InstanceToRelatedID(instance)
	rel.SetRelatedID(locale, ID)
	fmt.Println(rel.GetRelatedID(locale).Field().Name)
	fmt.Println(reflect.DeepEqual(rel.GetRelatedID(locale), IdOf(user)))
	// Output:
	// XqjMd3wdyUIWWM8N, XqjM4HwdyUY6z
	// ID
	// true
}

func ExampleRelationship_ParseRelatedID() {
	var user = &User{}
	user.ID.ParseString("XqjMd3wdyUIWWM8N")
	var locale = &UserLocale{}
	locale.ID.ParseString("XqjM4HwdyUY6z-AD")
	fmt.Println(user.ID, locale.ID)

	rel := StructOf(user).FieldsByName["Locales"].Relationship
	ID, _ := rel.ParseRelatedID("XqjMd3wdyUIWWM8N")
	fmt.Println(rel.GetRelatedID(user).Field().Name)
	rel.SetRelatedID(locale, ID)
	fmt.Println(rel.GetRelatedID(locale).Field().Name)
	fmt.Println(reflect.DeepEqual(rel.GetRelatedID(locale), IdOf(user)))
	// Output:
	// XqjMd3wdyUIWWM8N, XqjM4HwdyUY6z
	// ID
	// ID
	// true
}

func ExampleRelationship_SetRelatedID_fromSliceField() {
	var user = &User{}
	user.ID.ParseString("XqjMd3wdyUIWWM8N")
	var locale = &UserLocale{}
	locale.ID.ParseString("XqjM4HwdyUY6z-AD")
	fmt.Println(user.ID, locale.ID)

	rel := StructOf(user).FieldsByName["Locales"].Relationship
	rel.SetRelatedID(locale, IdOf(user))
	fmt.Println(rel.GetRelatedID(locale).Field().Name)
	fmt.Println(reflect.DeepEqual(rel.GetRelatedID(locale), IdOf(user)))
	// Output:
	// XqjMd3wdyUIWWM8N, XqjM4HwdyUY6z
	// ID
	// true
}

func ExampleRelationship_SetRelatedID_fromRelatedField() {
	var user = &User{}
	user.ID.ParseString("XqjMd3wdyUIWWM8N")
	var locale = &UserLocale{}
	locale.ID.ParseString("XqjM4HwdyUY6z-AD")
	fmt.Println(user.ID, locale.ID)

	rel := StructOf(locale).FieldsByName["User"].Relationship
	rel.SetRelatedID(locale, IdOf(user))
	fmt.Println(rel.GetRelatedID(locale).Field().Name)
	fmt.Println(reflect.DeepEqual(rel.GetRelatedID(locale), IdOf(user)))
	// Output:
	// XqjMd3wdyUIWWM8N, XqjM4HwdyUY6z
	// ID
	// true
}
