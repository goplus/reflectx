//go:build goplus.ifacefuncval && !llgo

package reflectx_test

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"

	"github.com/goplus/reflectx"
)

func TestIfaceFuncvalGC(t *testing.T) {
	styp := reflectx.NamedTypeOf("main", "GCInt", tyInt)
	typ := reflectx.NewMethodSet(styp, 1, 1)
	mString := reflectx.MakeMethod(
		"String",
		"main",
		false,
		reflect.FuncOf(nil, []reflect.Type{tyString}, false),
		func(args []reflect.Value) []reflect.Value {
			info := fmt.Sprintf("(%d)", args[0].Int())
			return []reflect.Value{reflect.ValueOf(info)}
		},
	)
	if err := reflectx.SetMethodSet(typ, []reflectx.Method{mString}, true); err != nil {
		t.Fatal(err)
	}

	values := make([]fmt.Stringer, 32)
	for i := range values {
		x := reflect.New(typ).Elem()
		x.SetInt(int64(i + 1))
		values[i] = x.Interface().(fmt.Stringer)
	}
	for i := 0; i < 10; i++ {
		runtime.GC()
	}
	for i, s := range values {
		want := fmt.Sprintf("(%d)", i+1)
		if got := s.String(); got != want {
			t.Fatalf("values[%d].String() = %q, want %q", i, got, want)
		}
		if got := fmt.Sprint(s); got != want {
			t.Fatalf("fmt.Sprint(values[%d]) = %q, want %q", i, got, want)
		}
	}
}
