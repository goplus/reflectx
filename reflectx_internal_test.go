package reflectx

import (
	"reflect"
	"testing"
)

func TestUnexportedHelpers(t *testing.T) {
	t1 := reflect.TypeOf(struct {
		A int
		B string
	}{})
	t2 := reflect.TypeOf(struct {
		A int
		B string
	}{})
	if !checkFields(t1, t2) {
		t.Fatal("identical")
	}
	if checkFields(t1, reflect.TypeOf(struct{ A int }{})) {
		t.Fatal("num field")
	}
	if checkFields(t1, reflect.TypeOf(struct {
		A int
		C string
	}{})) {
		t.Fatal("name")
	}
	if checkFields(t1, reflect.TypeOf(struct {
		A int8
		B string
	}{})) {
		t.Fatal("type")
	}
	anon := reflect.TypeOf(struct{ int }{})
	named := reflect.TypeOf(struct{ X int }{})
	if checkFields(anon, named) {
		t.Fatal("anonymous")
	}

	pt := reflect.PtrTo(t1)
	if toElem(pt) != t1 {
		t.Fatal("toElem ptr")
	}
	if toElem(t1) != t1 {
		t.Fatal("toElem")
	}
	v := reflect.New(t1)
	if toElemValue(v).Type() != t1 {
		t.Fatal("toElemValue ptr")
	}
	if toElemValue(v.Elem()).Type() != t1 {
		t.Fatal("toElemValue")
	}
	_ = isMethod(t1)

	n := newNameEx("x", "tag", false, true)
	setPkgPath(n, "main")
	setPkgPath(name{}, "main")
	_ = extraFieldMethod(0, reflect.TypeOf(helper(0)), map[string]bool{"String": true})
	_ = extraFieldMethod(0, reflect.PtrTo(reflect.TypeOf(helper(0))), nil)
}

type helper int

func (helper) String() string { return "" }

func (h *helper) Set(v int) { *h = helper(v) }
