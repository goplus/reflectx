// Tests here exercise functions defined only in !llgo-tagged source files (name.go, methodof.go).
//go:build !llgo

package reflectx

import (
	"reflect"
	"testing"
)

func TestNewNameExAndIsMethod(t *testing.T) {
	t1 := reflect.TypeOf(struct {
		A int
		B string
	}{})
	_ = isMethod(t1)

	n := newNameEx("x", "tag", false, true)
	setPkgPath(n, "main")
	setPkgPath(name{}, "main")
}
