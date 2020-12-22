package reflectx_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/goplus/reflectx"
)

var (
	byteTyp      = reflect.TypeOf(byte('a'))
	boolTyp      = reflect.TypeOf(true)
	intTyp       = reflect.TypeOf(0)
	strTyp       = reflect.TypeOf("")
	iType        = reflect.TypeOf((*interface{})(nil)).Elem()
	intSliceType = reflect.TypeOf([]int{})
)

func TestIntMethodOf(t *testing.T) {
	styp := reflectx.NamedTypeOf("main", "MyInt", intTyp)
	var typ reflect.Type
	mString := reflectx.MakeMethod(
		"String",
		false,
		reflect.FuncOf(nil, []reflect.Type{strTyp}, false),
		func(args []reflect.Value) []reflect.Value {
			v := args[0]
			info := fmt.Sprintf("(%d)", v.Int())
			return []reflect.Value{reflect.ValueOf(info)}
		},
	)
	mSet := reflectx.MakeMethod(
		"Set",
		true,
		reflect.FuncOf([]reflect.Type{intTyp}, nil, false),
		func(args []reflect.Value) (result []reflect.Value) {
			v := args[0].Elem()
			v.SetInt(args[1].Int())
			return
		},
	)
	mAppend := reflectx.MakeMethod(
		"Append",
		false,
		reflect.FuncOf([]reflect.Type{reflect.SliceOf(intTyp)}, []reflect.Type{intTyp}, true),
		func(args []reflect.Value) (result []reflect.Value) {
			var sum int64 = args[0].Int()
			for i := 0; i < args[1].Len(); i++ {
				sum += args[1].Index(i).Int()
			}
			return []reflect.Value{reflect.ValueOf(int(sum))}
		},
	)
	typ = reflectx.MethodOf(styp, []reflectx.Method{
		mString,
		mSet,
		mAppend,
	})
	ptrType := reflect.PtrTo(typ)

	if n := typ.NumMethod(); n != 2 {
		t.Fatal("typ.NumMethod()", n)
	}
	if n := ptrType.NumMethod(); n != 3 {
		t.Fatal("ptrTyp.NumMethod()", n)
	}

	pv := reflectx.New(typ).Elem()

	pv.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100)})

	if v := fmt.Sprint(reflectx.Interface(pv)); v != "(100)" {
		t.Fatalf("String(): have %v, want (100)", v)
	}
	if v := fmt.Sprint(reflectx.Interface(pv.Addr())); v != "(100)" {
		t.Fatalf("ptrTyp String(): have %v, want (100)", v)
	}

	// Append
	m0, _ := reflectx.MethodByName(typ, "Append")
	r0 := m0.Func.Call([]reflect.Value{pv, reflect.ValueOf(200), reflect.ValueOf(300), reflect.ValueOf(400)})
	if v := r0[0].Int(); v != 1000 {
		t.Fatalf("typ reflectx.MethodByName Testv: have %v, want 1000", v)
	}
	r0 = pv.MethodByName("Append").Call([]reflect.Value{reflect.ValueOf(200), reflect.ValueOf(300), reflect.ValueOf(400)})
	if v := r0[0].Int(); v != 1000 {
		t.Fatalf("typ value.MethodByName Testv: have %v, want 1000", v)
	}
}

func TestSliceMethodOf(t *testing.T) {
	styp := reflectx.NamedTypeOf("main", "IntSlice", reflect.TypeOf([]int{}))
	var typ reflect.Type
	mString := reflectx.MakeMethod(
		"String",
		false,
		reflect.FuncOf(nil, []reflect.Type{strTyp}, false),
		func(args []reflect.Value) []reflect.Value {
			v := args[0]
			info := fmt.Sprintf("(%v)", v.Convert(intSliceType))
			return []reflect.Value{reflect.ValueOf(info)}
		},
	)
	mSet := reflectx.MakeMethod(
		"Set",
		true,
		reflect.FuncOf([]reflect.Type{intSliceType}, nil, false),
		func(args []reflect.Value) (result []reflect.Value) {
			v := args[0].Elem()
			v.Set(args[1])
			return
		},
	)
	mAppend := reflectx.MakeMethod(
		"Append",
		false,
		reflect.FuncOf([]reflect.Type{reflect.SliceOf(intTyp)}, []reflect.Type{intTyp}, true),
		func(args []reflect.Value) (result []reflect.Value) {
			var sum int64
			for i := 0; i < args[0].Len(); i++ {
				sum += args[0].Index(i).Int()
			}
			for i := 0; i < args[1].Len(); i++ {
				sum += args[1].Index(i).Int()
			}
			return []reflect.Value{reflect.ValueOf(int(sum))}
		},
	)
	typ = reflectx.MethodOf(styp, []reflectx.Method{
		mString,
		mSet,
		mAppend,
	})
	ptrType := reflect.PtrTo(typ)

	if n := typ.NumMethod(); n != 2 {
		t.Fatal("typ.NumMethod()", n)
	}
	if n := ptrType.NumMethod(); n != 3 {
		t.Fatal("ptrTyp.NumMethod()", n)
	}

	pv := reflectx.New(typ).Elem()

	pv.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf([]int{100, 200, 300})})

	if v := fmt.Sprint(reflectx.Interface(pv)); v != "([100 200 300])" {
		t.Fatalf("String(): have %v, want ([100 200 300])", v)
	}
	if v := fmt.Sprint(reflectx.Interface(pv.Addr())); v != "([100 200 300])" {
		t.Fatalf("ptrTyp String(): have %v, want ([100 200 300])", v)
	}

	// Append
	m0, _ := reflectx.MethodByName(typ, "Append")
	r0 := m0.Func.Call([]reflect.Value{pv, reflect.ValueOf(200), reflect.ValueOf(300), reflect.ValueOf(400)})
	if v := r0[0].Int(); v != 1500 {
		t.Fatalf("typ reflectx.MethodByName Testv: have %v, want 1000", v)
	}
	r0 = pv.MethodByName("Append").Call([]reflect.Value{reflect.ValueOf(200), reflect.ValueOf(300), reflect.ValueOf(400)})
	if v := r0[0].Int(); v != 1500 {
		t.Fatalf("typ value.MethodByName Testv: have %v, want 1000", v)
	}
}

func TestArrayMethodOf(t *testing.T) {
	styp := reflectx.NamedTypeOf("main", "Point", reflect.TypeOf([2]int{}))
	var typ reflect.Type
	mString := reflectx.MakeMethod(
		"String",
		false,
		reflect.FuncOf(nil, []reflect.Type{strTyp}, false),
		func(args []reflect.Value) []reflect.Value {
			v := args[0]
			info := fmt.Sprintf("(%v,%v)", v.Index(0), v.Index(1))
			return []reflect.Value{reflect.ValueOf(info)}
		},
	)
	mSet := reflectx.MakeMethod(
		"Set",
		true,
		reflect.FuncOf([]reflect.Type{intTyp, intTyp}, nil, false),
		func(args []reflect.Value) (result []reflect.Value) {
			v := args[0].Elem()
			v.Index(0).Set(args[1])
			v.Index(1).Set(args[2])
			return
		},
	)
	mGet := reflectx.MakeMethod(
		"Get",
		false,
		reflect.FuncOf(nil, []reflect.Type{intTyp, intTyp}, false),
		func(args []reflect.Value) (result []reflect.Value) {
			v := args[0]
			return []reflect.Value{v.Index(0), v.Index(1)}
		},
	)
	typ = reflectx.MethodOf(styp, []reflectx.Method{
		mString,
		mSet,
		mGet,
	})
	ptrType := reflect.PtrTo(typ)

	if n := typ.NumMethod(); n != 2 {
		t.Fatal("typ.NumMethod()", n)
	}
	if n := ptrType.NumMethod(); n != 3 {
		t.Fatal("ptrTyp.NumMethod()", n)
	}

	pv := reflectx.New(typ).Elem()

	pv.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf(200)})

	if v := fmt.Sprint(reflectx.Interface(pv)); v != "(100,200)" {
		t.Fatalf("String(): have %v, want (100,200)", v)
	}
	if v := fmt.Sprint(reflectx.Interface(pv.Addr())); v != "(100,200)" {
		t.Fatalf("ptrTyp String(): have %v, want (100,200)", v)
	}

	// Get
	m0, _ := reflectx.MethodByName(typ, "Get")
	r0 := m0.Func.Call([]reflect.Value{pv})
	if len(r0) != 2 || r0[0].Int() != 100 || r0[1].Int() != 200 {
		t.Fatalf("typ reflectx.MethodByName Get: have %v, want 100 200", r0)
	}
	r0 = pv.MethodByName("Get").Call(nil)
	if len(r0) != 2 || r0[0].Int() != 100 || r0[1].Int() != 200 {
		t.Fatalf("typ value.MethodByName Get: have %v, want 100 200", r0)
	}
}

func TestStructMethodOf(t *testing.T) {
	fs := []reflect.StructField{
		reflect.StructField{Name: "X", Type: reflect.TypeOf(0)},
		reflect.StructField{Name: "Y", Type: reflect.TypeOf(0)},
	}
	styp := reflectx.NamedStructOf("main", "Point", fs)
	var typ reflect.Type
	mString := reflectx.MakeMethod(
		"String",
		false,
		reflect.FuncOf(nil, []reflect.Type{strTyp}, false),
		func(args []reflect.Value) []reflect.Value {
			v := args[0]
			info := fmt.Sprintf("(%v,%v)", v.Field(0), v.Field(1))
			return []reflect.Value{reflect.ValueOf(info)}
		},
	)
	mAdd := reflectx.MakeMethod(
		"Add",
		false,
		reflect.FuncOf([]reflect.Type{styp}, []reflect.Type{styp}, false),
		func(args []reflect.Value) []reflect.Value {
			v := reflectx.New(typ).Elem()
			v.Field(0).SetInt(args[0].Field(0).Int() + args[1].Field(0).Int())
			v.Field(1).SetInt(args[0].Field(1).Int() + args[1].Field(1).Int())
			return []reflect.Value{v}
		},
	)
	mSet := reflectx.MakeMethod(
		"Set",
		true,
		reflect.FuncOf([]reflect.Type{intTyp, intTyp}, nil, false),
		func(args []reflect.Value) (result []reflect.Value) {
			v := args[0].Elem()
			v.Field(0).Set(args[1])
			v.Field(1).Set(args[2])
			return
		},
	)
	mTestv := reflectx.MakeMethod(
		"Testv",
		false,
		reflect.FuncOf([]reflect.Type{reflect.SliceOf(intTyp)}, []reflect.Type{intTyp}, true),
		func(args []reflect.Value) (result []reflect.Value) {
			var sum int64 = args[0].Field(0).Int() + args[0].Field(1).Int()
			for i := 0; i < args[1].Len(); i++ {
				sum += args[1].Index(i).Int()
			}
			return []reflect.Value{reflect.ValueOf(int(sum))}
		},
	)
	typ = reflectx.MethodOf(styp, []reflectx.Method{
		mAdd,
		mString,
		mSet,
		mTestv,
	})
	ptrType := reflect.PtrTo(typ)

	if n := typ.NumMethod(); n != 3 {
		t.Fatal("typ.NumMethod()", n)
	}
	if n := ptrType.NumMethod(); n != 4 {
		t.Fatal("ptrTyp.NumMethod()", n)
	}

	pt1 := reflectx.New(typ).Elem()
	pt1.Field(0).SetInt(100)
	pt1.Field(1).SetInt(200)

	pt2 := reflectx.New(typ).Elem()
	pt2.Field(0).SetInt(300)
	pt2.Field(1).SetInt(400)

	// String
	if v := fmt.Sprint(reflectx.Interface(pt1)); v != "(100,200)" {
		t.Fatalf("String(): have %v, want (100,200)", v)
	}
	if v := fmt.Sprint(reflectx.Interface(pt1.Addr())); v != "(100,200)" {
		t.Fatalf("ptrTyp String(): have %v, want (100,200)", v)
	}

	// typ Add
	m, _ := reflectx.MethodByName(typ, "Add")
	r0 := m.Func.Call([]reflect.Value{pt1, pt2})
	if v := fmt.Sprint(reflectx.Interface(r0[0])); v != "(400,600)" {
		t.Fatalf("type reflectx.MethodByName Add: have %v, want (400,600)", v)
	}
	r0 = pt1.MethodByName("Add").Call([]reflect.Value{pt2})
	if v := fmt.Sprint(reflectx.Interface(r0[0])); v != "(400,600)" {
		t.Fatalf("value.MethodByName Add: have %v, want (400,600)", v)
	}

	// ptrtyp Add
	m, _ = reflectx.MethodByName(ptrType, "Add")
	r0 = m.Func.Call([]reflect.Value{pt1.Addr(), pt2})
	if v := fmt.Sprint(reflectx.Interface(r0[0])); v != "(400,600)" {
		t.Fatalf("ptrType reflectx.MethodByName Add: have %v, want (400,600)", v)
	}
	r0 = pt1.Addr().MethodByName("Add").Call([]reflect.Value{pt2})
	if v := fmt.Sprint(reflectx.Interface(r0[0])); v != "(400,600)" {
		t.Fatalf("ptrType value.reflectx.MethodByName Add: have %v, want (400,600)", v)
	}

	// Set
	m0, _ := reflectx.MethodByName(ptrType, "Set")
	m0.Func.Call([]reflect.Value{pt1.Addr(), reflect.ValueOf(-100), reflect.ValueOf(-200)})
	if v := fmt.Sprint(reflectx.Interface(pt1)); v != "(-100,-200)" {
		t.Fatalf("ptrType reflectx.MethodByName Set: have %v, want (-100,-200)", v)
	}
	pt1.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(1), reflect.ValueOf(2)})
	if v := fmt.Sprint(reflectx.Interface(pt1)); v != "(1,2)" {
		t.Fatalf("ptrType reflectx.MethodByName Set: have %v, want (1,2)", v)
	}

	// Testv
	m0, _ = reflectx.MethodByName(typ, "Testv")
	r0 = m0.Func.Call([]reflect.Value{pt2, reflect.ValueOf(200), reflect.ValueOf(300), reflect.ValueOf(400)})
	if v := r0[0].Int(); v != 1600 {
		t.Fatalf("typ reflectx.MethodByName Testv: have %v, want 1600", v)
	}
	r0 = pt2.MethodByName("Testv").Call([]reflect.Value{reflect.ValueOf(200), reflect.ValueOf(300), reflect.ValueOf(400)})
	if v := r0[0].Int(); v != 1600 {
		t.Fatalf("typ value.MethodByName Testv: have %v, want 1600", v)
	}
}
