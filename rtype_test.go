//go:build !llgo

package reflectx_test

import (
	"reflect"
	"testing"

	"github.com/goplus/reflectx"
)

func TestStructCloneFieldPkgPath(t *testing.T) {
	types := []struct {
		name string
		typ  reflect.Type
	}{
		{"anonymous", reflect.TypeOf(struct {
			private int `test:"private"`
			Public  string
		}{})},
		{"embedded", reflect.TypeOf(struct {
			nPoint
			Public string
		}{})},
		{"dynamic", reflect.StructOf([]reflect.StructField{
			{Name: "private", PkgPath: "example/model", Type: tyInt},
			{Name: "Public", Type: tyInt},
		})},
		{"named", reflect.TypeOf(nPoint{})},
	}
	cloners := []struct {
		name string
		fn   func(reflect.Type) reflect.Type
	}{
		{"NewMethodSet", func(typ reflect.Type) reflect.Type {
			return reflectx.NewContext().NewMethodSet(typ, 0, 1)
		}},
		{"NamedTypeOf", func(typ reflect.Type) reflect.Type {
			return reflectx.NamedTypeOf(typ.PkgPath(), typ.Name(), typ)
		}},
	}
	for _, source := range types {
		for _, cloner := range cloners {
			t.Run(source.name+"/"+cloner.name, func(t *testing.T) {
				cloned := cloner.fn(source.typ)
				if cloned.Name() != source.typ.Name() || cloned.PkgPath() != source.typ.PkgPath() {
					t.Fatal("cloning changed the type name or package path")
				}
				for i := 0; i < source.typ.NumField(); i++ {
					want, got := source.typ.Field(i), cloned.Field(i)
					if !reflect.DeepEqual(got, want) {
						t.Errorf("field %d: got %+v, want %+v", i, got, want)
					}
				}
			})
		}
	}
}
