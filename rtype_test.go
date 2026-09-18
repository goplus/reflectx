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

type clonePrivateInterface interface {
	private() int
	Public() int
}

type clonePrivateValue struct{}

func (clonePrivateValue) private() int { return 42 }
func (clonePrivateValue) Public() int  { return 7 }

func TestInterfaceCloneMethodPkgPath(t *testing.T) {
	value := reflect.ValueOf(clonePrivateValue{})
	types := []struct {
		name string
		typ  reflect.Type
	}{
		{"named", reflect.TypeOf((*clonePrivateInterface)(nil)).Elem()},
		{"anonymous", reflect.TypeOf((*interface {
			private() int
			Public() int
		})(nil)).Elem()},
		{"dynamic", reflectx.NewContext().InterfaceOf(nil, []reflect.Method{
			{Name: "private", PkgPath: value.Type().PkgPath(), Type: reflect.TypeOf(func() int { return 0 })},
			{Name: "Public", Type: reflect.TypeOf(func() int { return 0 })},
		})},
		{"exported", reflect.TypeOf((*interface{ Public() int })(nil)).Elem()},
	}
	for _, source := range types {
		t.Run(source.name, func(t *testing.T) {
			cloned := reflectx.NamedTypeOf("", "", source.typ)
			if cloned.NumMethod() != source.typ.NumMethod() {
				t.Fatal("cloning changed the method count")
			}
			for i := 0; i < source.typ.NumMethod(); i++ {
				want, got := source.typ.Method(i), cloned.Method(i)
				if got != want {
					t.Errorf("method %d: got %+v, want %+v", i, got, want)
				}
			}
			if !source.typ.Implements(cloned) || !cloned.Implements(source.typ) {
				t.Error("cloned interface no longer matches its source")
			}
			if !value.Type().Implements(cloned) {
				t.Fatal("concrete value no longer implements the cloned interface")
			}
			dst := reflect.New(cloned).Elem()
			dst.Set(value)
			if got := dst.MethodByName("Public").Call(nil)[0].Int(); got != 7 {
				t.Fatalf("Public() = %d, want 7", got)
			}
		})
	}
}
