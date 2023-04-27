package reflectx_test

import (
	"fmt"
	"io"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/goplus/reflectx"
)

var (
	tyByte           = reflect.TypeOf(byte('a'))
	tyBool           = reflect.TypeOf(true)
	tyInt            = reflect.TypeOf(0)
	tyString         = reflect.TypeOf("")
	tyError          = reflect.TypeOf((*error)(nil)).Elem()
	tyEmptyStruct    = reflect.TypeOf((*struct{})(nil)).Elem()
	tyEmptyInterface = reflect.TypeOf((*interface{})(nil)).Elem()
	emtpyStruct      struct{}
)

type Int int

func (i Int) String() string {
	return fmt.Sprintf("(%v)", int(i))
}

func (i *Int) Set(v int) {
	*(*int)(i) = v
}

func (i Int) Append(v ...int) int {
	sum := int(i)
	for _, n := range v {
		sum += n
	}
	return sum
}

func TestIntMethodOf(t *testing.T) {
	// Int type
	var i Int
	i.Set(100)
	if v := fmt.Sprint(i); v != "(100)" {
		t.Fatalf("String(): have %v, want (100)", v)
	}
	if v := i.Append(200, 300, 400); v != 1000 {
		t.Fatalf("Append(): have %v, want (1000)", v)
	}
	// make Int type
	styp := reflectx.NamedTypeOf("main", "Int", tyInt)
	typ := reflectx.NewMethodSet(styp, 2, 3)
	mString := reflectx.MakeMethod(
		"String",
		"main",
		false,
		reflect.FuncOf(nil, []reflect.Type{tyString}, false),
		func(args []reflect.Value) []reflect.Value {
			v := args[0]
			info := fmt.Sprintf("(%d)", v.Int())
			return []reflect.Value{reflect.ValueOf(info)}
		},
	)
	mSet := reflectx.MakeMethod(
		"Set",
		"main",
		true,
		reflect.FuncOf([]reflect.Type{tyInt}, nil, false),
		func(args []reflect.Value) (result []reflect.Value) {
			v := args[0].Elem()
			v.SetInt(args[1].Int())
			return
		},
	)
	mAppend := reflectx.MakeMethod(
		"Append",
		"main",
		false,
		reflect.FuncOf([]reflect.Type{reflect.SliceOf(tyInt)}, []reflect.Type{tyInt}, true),
		func(args []reflect.Value) (result []reflect.Value) {
			var sum int64 = args[0].Int()
			for i := 0; i < args[1].Len(); i++ {
				sum += args[1].Index(i).Int()
			}
			return []reflect.Value{reflect.ValueOf(int(sum))}
		},
	)
	reflectx.SetMethodSet(typ, []reflectx.Method{
		mString,
		mSet,
		mAppend,
	}, true)
	ptrType := reflect.PtrTo(typ)

	if n := typ.NumMethod(); n != 2 {
		t.Fatal("typ.NumMethod()", n)
	}
	if n := ptrType.NumMethod(); n != 3 {
		t.Fatal("ptrTyp.NumMethod()", n)
	}

	x := reflect.New(typ).Elem()
	x.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100)})

	// String
	if v := fmt.Sprint(x); v != "(100)" {
		t.Fatalf("String(): have %v, want (100)", v)
	}
	if v := fmt.Sprint(x.Addr()); v != "(100)" {
		t.Fatalf("ptrTyp String(): have %v, want (100)", v)
	}

	// Append
	m, _ := reflectx.MethodByName(typ, "Append")
	r := m.Func.Call([]reflect.Value{x, reflect.ValueOf(200), reflect.ValueOf(300), reflect.ValueOf(400)})
	if v := r[0].Int(); v != 1000 {
		t.Fatalf("typ reflectx.MethodByName Append: have %v, want 1000", v)
	}
	r = x.MethodByName("Append").Call([]reflect.Value{reflect.ValueOf(200), reflect.ValueOf(300), reflect.ValueOf(400)})
	if v := r[0].Int(); v != 1000 {
		t.Fatalf("typ value.MethodByName Append: have %v, want 1000", v)
	}
}

type IntSlice []int

func (i IntSlice) String() string {
	return fmt.Sprintf("{%v}%v", len(i), ([]int)(i))
}

func (i *IntSlice) Set(v ...int) {
	*i = v
}

func (i IntSlice) Append(v ...int) int {
	var sum int
	for _, n := range i {
		sum += n
	}
	for _, n := range v {
		sum += n
	}
	return sum
}

func TestSliceMethodOf(t *testing.T) {
	// IntSlice type
	var i IntSlice
	i.Set(100, 200, 300)
	if v := i.String(); v != "{3}[100 200 300]" {
		t.Fatalf("have %v, want {3}[100 200 300]", v)
	}
	if v := i.Append(200, 300, 400); v != 1500 {
		t.Fatalf("have %v, want 1500", v)
	}
	// make IntSlice type
	intSliceTyp := reflect.TypeOf([]int{})
	styp := reflectx.NamedTypeOf("main", "IntSlice", intSliceTyp)
	typ := reflectx.NewMethodSet(styp, 2, 3)
	mString := reflectx.MakeMethod(
		"String",
		"main",
		false,
		reflect.FuncOf(nil, []reflect.Type{tyString}, false),
		func(args []reflect.Value) []reflect.Value {
			v := args[0]
			info := fmt.Sprintf("{%v}%v", v.Len(), v.Convert(intSliceTyp))
			return []reflect.Value{reflect.ValueOf(info)}
		},
	)
	mSet := reflectx.MakeMethod(
		"Set",
		"main",
		true,
		reflect.FuncOf([]reflect.Type{intSliceTyp}, nil, true),
		func(args []reflect.Value) (result []reflect.Value) {
			v := args[0].Elem()
			v.Set(args[1])
			return
		},
	)
	mAppend := reflectx.MakeMethod(
		"Append",
		"main",
		false,
		reflect.FuncOf([]reflect.Type{reflect.SliceOf(tyInt)}, []reflect.Type{tyInt}, true),
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
	reflectx.SetMethodSet(typ, []reflectx.Method{
		mString,
		mSet,
		mAppend,
	}, true)
	ptrType := reflect.PtrTo(typ)

	if n := typ.NumMethod(); n != 2 {
		t.Fatal("typ.NumMethod()", n)
	}
	if n := ptrType.NumMethod(); n != 3 {
		t.Fatal("ptrTyp.NumMethod()", n)
	}

	x := reflect.New(typ).Elem()
	x.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf(200), reflect.ValueOf(300)})

	// String
	if v := fmt.Sprint(x); v != "{3}[100 200 300]" {
		t.Fatalf("String(): have %v, want {3}[100 200 300]", v)
	}
	if v := fmt.Sprint(x.Addr()); v != "{3}[100 200 300]" {
		t.Fatalf("ptrTyp String(): have %v, want {3}[100 200 300]", v)
	}

	// Append
	m, _ := reflectx.MethodByName(typ, "Append")
	r := m.Func.Call([]reflect.Value{x, reflect.ValueOf(200), reflect.ValueOf(300), reflect.ValueOf(400)})
	if v := r[0].Int(); v != 1500 {
		t.Fatalf("typ reflectx.MethodByName Append: have %v, want 1000", v)
	}
	r = x.MethodByName("Append").Call([]reflect.Value{reflect.ValueOf(200), reflect.ValueOf(300), reflect.ValueOf(400)})
	if v := r[0].Int(); v != 1500 {
		t.Fatalf("typ value.MethodByName Append: have %v, want 1000", v)
	}
}

type IntArray [2]int

func (i IntArray) String() string {
	return fmt.Sprintf("(%v,%v)", i[0], i[1])
}

func (i *IntArray) Set(x, y int) {
	*(*int)(&i[0]), *(*int)(&i[1]) = x, y
}

func (i IntArray) Get() (int, int) {
	return i[0], i[1]
}

func (i IntArray) Scale(v int) IntArray {
	return IntArray{i[0] * v, i[1] * v}
}

func TestArrayMethodOf(t *testing.T) {
	if runtime.Compiler == "gopherjs" {
		t.Skip("skip gopherjs")
	}
	// IntArray
	var i IntArray
	i.Set(100, 200)
	if v := fmt.Sprint(i); v != "(100,200)" {
		t.Fatalf("have %v, want (100,200)", v)
	}
	if v1, v2 := i.Get(); v1 != 100 || v2 != 200 {
		t.Fatalf("have %v %v, want 100 200)", v1, v2)
	}
	if v := fmt.Sprint(i.Scale(5)); v != "(500,1000)" {
		t.Fatalf("have %v, want (500,1000)", v)
	}
	styp := reflectx.NamedTypeOf("main", "IntArray", reflect.TypeOf([2]int{}))
	// make IntArray
	typ := reflectx.NewMethodSet(styp, 3, 4)

	mString := reflectx.MakeMethod(
		"String",
		"main",
		false,
		reflect.FuncOf(nil, []reflect.Type{tyString}, false),
		func(args []reflect.Value) []reflect.Value {
			v := args[0]
			info := fmt.Sprintf("(%v,%v)", v.Index(0), v.Index(1))
			return []reflect.Value{reflect.ValueOf(info)}
		},
	)
	mSet := reflectx.MakeMethod(
		"Set",
		"main",
		true,
		reflect.FuncOf([]reflect.Type{tyInt, tyInt}, nil, false),
		func(args []reflect.Value) (result []reflect.Value) {
			v := args[0].Elem()
			v.Index(0).Set(args[1])
			v.Index(1).Set(args[2])
			return
		},
	)
	mGet := reflectx.MakeMethod(
		"Get",
		"main",
		false,
		reflect.FuncOf(nil, []reflect.Type{tyInt, tyInt}, false),
		func(args []reflect.Value) (result []reflect.Value) {
			v := args[0]
			return []reflect.Value{v.Index(0), v.Index(1)}
		},
	)
	mScale := reflectx.MakeMethod(
		"Scale",
		"main",
		false,
		reflect.FuncOf([]reflect.Type{tyInt}, []reflect.Type{typ}, false),
		func(args []reflect.Value) (result []reflect.Value) {
			v := args[0]
			s := args[1].Int()
			r := reflect.New(typ).Elem()
			r.Index(0).SetInt(v.Index(0).Int() * s)
			r.Index(1).SetInt(v.Index(1).Int() * s)
			return []reflect.Value{r}
		},
	)
	reflectx.SetMethodSet(typ, []reflectx.Method{
		mString,
		mSet,
		mGet,
		mScale,
	}, true)
	ptrType := reflect.PtrTo(typ)

	if n := typ.NumMethod(); n != 3 {
		t.Fatal("typ.NumMethod()", n)
	}
	if n := ptrType.NumMethod(); n != 4 {
		t.Fatal("ptrTyp.NumMethod()", n)
	}

	x := reflect.New(typ).Elem()
	x.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf(200)})

	// String
	if v := fmt.Sprint(x); v != "(100,200)" {
		t.Fatalf("String(): have %v, want (100,200)", v)
	}
	if v := fmt.Sprint(x.Addr()); v != "(100,200)" {
		t.Fatalf("ptrTyp String(): have %v, want (100,200)", v)
	}

	// Get
	m, _ := reflectx.MethodByName(typ, "Get")
	r := m.Func.Call([]reflect.Value{x})
	if len(r) != 2 || r[0].Int() != 100 || r[1].Int() != 200 {
		t.Fatalf("typ reflectx.MethodByName Get: have %v, want 100 200", r)
	}
	r = x.MethodByName("Get").Call(nil)
	if len(r) != 2 || r[0].Int() != 100 || r[1].Int() != 200 {
		t.Fatalf("typ value.MethodByName Get: have %v, want 100 200", r)
	}

	// Scale
	m, _ = reflectx.MethodByName(typ, "Scale")
	r = m.Func.Call([]reflect.Value{x, reflect.ValueOf(5)})
	if v := fmt.Sprint(r[0]); v != "(500,1000)" {
		t.Fatalf("typ reflectx.MethodByName Scale: have %v, want (500,1000)", v)
	}
	r = x.MethodByName("Scale").Call([]reflect.Value{reflect.ValueOf(5)})
	if v := fmt.Sprint((r[0])); v != "(500,1000)" {
		t.Fatalf("typ value.MethodByName Scale: have %v, want (500,1000)", v)
	}
}

type Point struct {
	X int
	Y int
}

func (i Point) String() string {
	return fmt.Sprintf("(%v,%v)", i.X, i.Y)
}

func (i Point) Add(v Point) Point {
	return Point{i.X + v.X, i.Y + v.Y}
}

func (i *Point) Set(x, y int) {
	i.X, i.Y = x, y
}

func (i Point) Scale(v ...int) (ar []Point) {
	for _, n := range v {
		ar = append(ar, Point{i.X * n, i.Y * n})
	}
	return
}

func (i Point) New() *Point {
	return &Point{i.X, i.Y}
}

func makeDynamicPointType() reflect.Type {
	fs := []reflect.StructField{
		reflect.StructField{Name: "X", Type: reflect.TypeOf(0)},
		reflect.StructField{Name: "Y", Type: reflect.TypeOf(0)},
	}
	styp := reflectx.NamedStructOf("main", "Point", fs)
	//var typ reflect.Type
	typ := reflectx.NewMethodSet(styp, 4, 5)
	mString := reflectx.MakeMethod(
		"String",
		"main",
		false,
		reflect.FuncOf(nil, []reflect.Type{tyString}, false),
		func(args []reflect.Value) []reflect.Value {
			v := args[0]
			info := fmt.Sprintf("(%v,%v)", v.Field(0), v.Field(1))
			return []reflect.Value{reflect.ValueOf(info)}
		},
	)
	mAdd := reflectx.MakeMethod(
		"Add",
		"main",
		false,
		reflect.FuncOf([]reflect.Type{typ}, []reflect.Type{typ}, false),
		func(args []reflect.Value) []reflect.Value {
			v := reflect.New(typ).Elem()
			v.Field(0).SetInt(args[0].Field(0).Int() + args[1].Field(0).Int())
			v.Field(1).SetInt(args[0].Field(1).Int() + args[1].Field(1).Int())
			return []reflect.Value{v}
		},
	)
	mSet := reflectx.MakeMethod(
		"Set",
		"main",
		true,
		reflect.FuncOf([]reflect.Type{tyInt, tyInt}, nil, false),
		func(args []reflect.Value) (result []reflect.Value) {
			v := args[0].Elem()
			v.Field(0).Set(args[1])
			v.Field(1).Set(args[2])
			return
		},
	)
	mScale := reflectx.MakeMethod(
		"Scale",
		"main",
		false,
		reflect.FuncOf([]reflect.Type{reflect.SliceOf(tyInt)}, []reflect.Type{reflect.SliceOf(typ)}, true),
		func(args []reflect.Value) (result []reflect.Value) {
			x, y := args[0].Field(0).Int(), args[0].Field(1).Int()
			r := reflect.MakeSlice(reflect.SliceOf(typ), 0, 0)
			for i := 0; i < args[1].Len(); i++ {
				s := args[1].Index(i).Int()
				v := reflect.New(typ).Elem()
				v.Field(0).SetInt(x * s)
				v.Field(1).SetInt(y * s)
				r = reflect.Append(r, v)
			}
			return []reflect.Value{r}
		},
	)
	mNew := reflectx.MakeMethod(
		"New",
		"main",
		false,
		reflect.FuncOf(nil, []reflect.Type{reflect.PtrTo(typ)}, false),
		func(args []reflect.Value) (result []reflect.Value) {
			v := reflect.New(typ).Elem()
			v.Field(0).SetInt(args[0].Field(0).Int())
			v.Field(1).SetInt(args[0].Field(1).Int())
			return []reflect.Value{v.Addr()}
		},
	)
	reflectx.SetMethodSet(typ, []reflectx.Method{
		mAdd,
		mString,
		mSet,
		mScale,
		mNew,
	}, true)
	return typ
}

func TestStructMethodOf(t *testing.T) {
	// Point
	var i Point
	i.Set(100, 200)
	if v := fmt.Sprint(i); v != "(100,200)" {
		t.Fatalf("have %v, want (100,200)", v)
	}
	if v := fmt.Sprint(i.Add(Point{1, 2})); v != "(101,202)" {
		t.Fatalf("have %v, want (101,202)", v)
	}
	if v := fmt.Sprint(i.Scale(2, 3, 4)); v != "[(200,400) (300,600) (400,800)]" {
		t.Fatalf("have %v, want [(200,400) (300,600) (400,800)]", v)
	}
	if v := fmt.Sprint(i.New()); v != "(100,200)" {
		t.Fatalf("have %v, want (100,200)", v)
	}
	// make Point
	typ := makeDynamicPointType()
	ptrType := reflect.PtrTo(typ)

	if n := typ.NumMethod(); n != 4 {
		t.Fatal("typ.NumMethod()", n)
	}
	if n := ptrType.NumMethod(); n != 5 {
		t.Fatal("ptrTyp.NumMethod()", n)
	}

	pt1 := reflect.New(typ).Elem()
	pt1.Field(0).SetInt(100)
	pt1.Field(1).SetInt(200)

	pt2 := reflect.New(typ).Elem()
	pt2.Field(0).SetInt(300)
	pt2.Field(1).SetInt(400)

	// String
	if v := fmt.Sprint(pt1); v != "(100,200)" {
		t.Fatalf("String(): have %v, want (100,200)", v)
	}
	if v := fmt.Sprint(pt1.Addr()); v != "(100,200)" {
		t.Fatalf("ptrTyp String(): have %v, want (100,200)", v)
	}

	// typ Add
	m, _ := reflectx.MethodByName(typ, "Add")
	r := m.Func.Call([]reflect.Value{pt1, pt2})
	if v := fmt.Sprint(r[0]); v != "(400,600)" {
		t.Fatalf("type reflectx.MethodByName Add: have %v, want (400,600)", v)
	}
	r = pt1.MethodByName("Add").Call([]reflect.Value{pt2})
	if v := fmt.Sprint(r[0]); v != "(400,600)" {
		t.Fatalf("value.MethodByName Add: have %v, want (400,600)", v)
	}

	// ptrtyp Add
	m, _ = reflectx.MethodByName(ptrType, "Add")
	r = m.Func.Call([]reflect.Value{pt1.Addr(), pt2})
	if v := fmt.Sprint(r[0]); v != "(400,600)" {
		t.Fatalf("ptrType reflectx.MethodByName Add: have %v, want (400,600)", v)
	}
	r = pt1.Addr().MethodByName("Add").Call([]reflect.Value{pt2})
	if v := fmt.Sprint(r[0]); v != "(400,600)" {
		t.Fatalf("ptrType value.reflectx.MethodByName Add: have %v, want (400,600)", v)
	}

	// Set
	m, _ = reflectx.MethodByName(ptrType, "Set")
	m.Func.Call([]reflect.Value{pt1.Addr(), reflect.ValueOf(-100), reflect.ValueOf(-200)})
	if v := fmt.Sprint(pt1); v != "(-100,-200)" {
		t.Fatalf("ptrType reflectx.MethodByName Set: have %v, want (-100,-200)", v)
	}
	pt1.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf(200)})
	if v := fmt.Sprint(pt1); v != "(100,200)" {
		t.Fatalf("ptrType reflectx.MethodByName Set: have %v, want (100,200)", v)
	}

	// Scale
	m, _ = reflectx.MethodByName(typ, "Scale")
	r = m.Func.Call([]reflect.Value{pt1, reflect.ValueOf(2), reflect.ValueOf(3), reflect.ValueOf(4)})
	if v := fmt.Sprint(v2is(r[0])); v != "[(200,400) (300,600) (400,800)]" {
		t.Fatalf("have %v, want [(200,400) (300,600) (400,800)]", v)
	}
	r = pt1.MethodByName("Scale").Call([]reflect.Value{reflect.ValueOf(2), reflect.ValueOf(3), reflect.ValueOf(4)})
	if v := fmt.Sprint(v2is(r[0])); v != "[(200,400) (300,600) (400,800)]" {
		t.Fatalf("have %v, want [(200,400) (300,600) (400,800)]", v)
	}

	// New
	m, _ = reflectx.MethodByName(typ, "New")
	r = m.Func.Call([]reflect.Value{pt1})
	if v := fmt.Sprint(r[0]); v != "(100,200)" {
		t.Fatalf("have %v, want (100,200)", v)
	}
	r = pt1.MethodByName("New").Call(nil)
	if v := fmt.Sprint(r[0]); v != "(100,200)" {
		t.Fatalf("have %v, want (100,200)", v)
	}
}

func v2is(v reflect.Value) (is []interface{}) {
	for i := 0; i < v.Len(); i++ {
		is = append(is, v.Index(i).Interface())
	}
	return is
}

type testMethodStack struct {
	name    string
	pkgpath string
	mtyp    reflect.Type
	fun     func([]reflect.Value) []reflect.Value
	args    []reflect.Value
	result  []reflect.Value
	pointer bool
}

var (
	testMethodStacks = []testMethodStack{
		testMethodStack{
			"Empty",
			"main",
			reflect.FuncOf(nil, nil, false),
			func(args []reflect.Value) []reflect.Value {
				if len(args) != 1 {
					panic(fmt.Errorf("args have %v, want nil", args[1:]))
				}
				return nil
			},
			nil,
			nil,
			false,
		},
		testMethodStack{
			"Empty Struct",
			"main",
			reflect.FuncOf([]reflect.Type{tyEmptyStruct}, []reflect.Type{tyEmptyStruct}, false),
			func(args []reflect.Value) []reflect.Value {
				return []reflect.Value{args[1]}
			},
			[]reflect.Value{reflect.ValueOf(emtpyStruct)},
			[]reflect.Value{reflect.ValueOf(emtpyStruct)},
			false,
		},
		testMethodStack{
			"Empty Struct2",
			"main",
			reflect.FuncOf([]reflect.Type{tyEmptyStruct, tyInt, tyEmptyStruct}, []reflect.Type{tyEmptyStruct, tyInt, tyEmptyStruct}, false),
			func(args []reflect.Value) []reflect.Value {
				return []reflect.Value{args[1], args[2], args[3]}
			},
			[]reflect.Value{reflect.ValueOf(emtpyStruct), reflect.ValueOf(100), reflect.ValueOf(emtpyStruct)},
			[]reflect.Value{reflect.ValueOf(emtpyStruct), reflect.ValueOf(100), reflect.ValueOf(emtpyStruct)},
			false,
		},
		testMethodStack{
			"Empty Struct3",
			"main",
			reflect.FuncOf([]reflect.Type{tyEmptyStruct, tyEmptyStruct, tyInt, tyEmptyStruct}, []reflect.Type{tyInt}, false),
			func(args []reflect.Value) []reflect.Value {
				return []reflect.Value{args[3]}
			},
			[]reflect.Value{reflect.ValueOf(emtpyStruct), reflect.ValueOf(emtpyStruct), reflect.ValueOf(100), reflect.ValueOf(emtpyStruct)},
			[]reflect.Value{reflect.ValueOf(100)},
			false,
		},
		testMethodStack{
			"Empty Struct4",
			"main",
			reflect.FuncOf([]reflect.Type{tyEmptyStruct, tyEmptyStruct, tyInt, tyEmptyStruct}, []reflect.Type{tyEmptyStruct, tyEmptyStruct, tyEmptyStruct, tyBool}, false),
			func(args []reflect.Value) []reflect.Value {
				return []reflect.Value{reflect.ValueOf(emtpyStruct), reflect.ValueOf(emtpyStruct), reflect.ValueOf(emtpyStruct), reflect.ValueOf(true)}
			},
			[]reflect.Value{reflect.ValueOf(emtpyStruct), reflect.ValueOf(emtpyStruct), reflect.ValueOf(100), reflect.ValueOf(emtpyStruct)},
			[]reflect.Value{reflect.ValueOf(emtpyStruct), reflect.ValueOf(emtpyStruct), reflect.ValueOf(emtpyStruct), reflect.ValueOf(true)},
			false,
		},
		testMethodStack{
			"Bool_Nil",
			"main",
			reflect.FuncOf([]reflect.Type{tyBool}, nil, false),
			func(args []reflect.Value) []reflect.Value {
				return nil
			},
			[]reflect.Value{reflect.ValueOf(true)},
			nil,
			false,
		},
		testMethodStack{
			"Bool_Bool",
			"main",
			reflect.FuncOf([]reflect.Type{tyBool}, []reflect.Type{tyBool}, false),
			func(args []reflect.Value) []reflect.Value {
				return []reflect.Value{args[1]}
			},
			[]reflect.Value{reflect.ValueOf(true)},
			[]reflect.Value{reflect.ValueOf(true)},
			false,
		},
		testMethodStack{
			"Int_Int",
			"main",
			reflect.FuncOf([]reflect.Type{tyInt}, []reflect.Type{tyInt}, false),
			func(args []reflect.Value) []reflect.Value {
				v := 300 + args[1].Int()
				return []reflect.Value{reflect.ValueOf(int(v))}
			},
			[]reflect.Value{reflect.ValueOf(-200)},
			[]reflect.Value{reflect.ValueOf(100)},
			false,
		},
		testMethodStack{
			"Big Bytes_ByteInt",
			"main",
			reflect.FuncOf([]reflect.Type{reflect.TypeOf([4096]byte{})}, []reflect.Type{tyByte, tyInt, tyByte}, false),
			func(args []reflect.Value) []reflect.Value {
				return []reflect.Value{args[1].Index(1), reflect.ValueOf(args[1].Len()), args[1].Index(3)}
			},
			[]reflect.Value{reflect.ValueOf([4096]byte{'a', 'b', 'c', 'd', 'e'})},
			[]reflect.Value{reflect.ValueOf('b'), reflect.ValueOf(4096), reflect.ValueOf('d')},
			true,
		},
	}
)

func TestMethodStack(t *testing.T) {
	// make Point
	fs := []reflect.StructField{
		reflect.StructField{Name: "X", Type: reflect.TypeOf(0)},
		reflect.StructField{Name: "Y", Type: reflect.TypeOf(0)},
	}
	styp := reflectx.NamedStructOf("main", "Point", fs)
	typ := reflectx.NewMethodSet(styp, len(testMethodStacks), len(testMethodStacks))
	var methods []reflectx.Method
	for _, m := range testMethodStacks {
		mm := reflectx.MakeMethod(
			m.name,
			m.pkgpath,
			m.pointer,
			m.mtyp,
			m.fun,
		)
		methods = append(methods, mm)
	}
	reflectx.SetMethodSet(typ, methods, true)
	v := reflect.New(typ).Elem()
	v.Field(0).SetInt(100)
	v.Field(1).SetInt(200)
	for _, m := range testMethodStacks {
		var r []reflect.Value
		if m.pointer {
			r = v.Addr().MethodByName(m.name).Call(m.args)
		} else {
			r = v.MethodByName(m.name).Call(m.args)
		}
		if len(r) != len(m.result) {
			t.Fatalf("failed %v %v, have %v want %v", m.name, m.mtyp, r, m.result)
		}
		for i := 0; i < len(r); i++ {
			if fmt.Sprint(r[i]) != fmt.Sprint(m.result[i]) {
				t.Fatalf("failed %v, have %v want %v", m.name, r[i], m.result[i])
			}
		}
	}
}

func checkInterface(t *testing.T, typ, styp reflect.Type) {
	if typ.NumMethod() != styp.NumMethod() {
		t.Errorf("num method: have %v, want %v", typ.NumMethod(), styp.NumMethod())
	}
	for i := 0; i < typ.NumMethod(); i++ {
		if typ.Method(i) != styp.Method(i) {
			t.Errorf("method: have %v, want %v", typ.Method(i), styp.Method(i))
		}
	}
	if !typ.ConvertibleTo(styp) {
		t.Errorf("%v cannot ConvertibleTo %v", typ, styp)
	}
	if !styp.ConvertibleTo(typ) {
		t.Errorf("%v cannot ConvertibleTo %v", styp, typ)
	}
}

func TestInterfaceOf(t *testing.T) {
	ms := []reflect.Method{
		reflect.Method{
			Name: "String",
			Type: reflect.FuncOf(nil, []reflect.Type{tyString}, false),
		},
		reflect.Method{
			Name: "Test",
			Type: reflect.FuncOf(nil, []reflect.Type{tyBool}, false),
		},
	}
	typ1 := reflectx.InterfaceOf(nil, ms)
	typ2 := reflectx.InterfaceOf(nil, ms)
	if typ1 != typ2 {
		t.Fatalf("different type: %v %v", typ1, typ2)
	}
}

func TestNamedInterfaceOf(t *testing.T) {
	pkgpath := "github.com/goplus/reflectx"
	typ := reflectx.NamedInterfaceOf(pkgpath, "Stringer", nil,
		[]reflect.Method{
			reflect.Method{
				Name: "String",
				Type: reflect.FuncOf(nil, []reflect.Type{tyString}, false),
			},
		},
	)
	checkInterface(t, typ, reflect.TypeOf((*fmt.Stringer)(nil)).Elem())

	typ = reflectx.NamedInterfaceOf(pkgpath, "ReadWriteCloser",
		[]reflect.Type{
			reflect.TypeOf((*io.Reader)(nil)).Elem(),
			reflect.TypeOf((*io.Writer)(nil)).Elem(),
		},
		[]reflect.Method{
			reflect.Method{
				Name: "Close",
				Type: reflect.FuncOf(nil, []reflect.Type{tyError}, false),
			},
		},
	)
	checkInterface(t, typ, reflect.TypeOf((*io.ReadWriteCloser)(nil)).Elem())
}

func TestNamedInterfaceOf2(t *testing.T) {
	pkgpath := "github.com/goplus/reflectx"
	typ := reflectx.NewInterfaceType(pkgpath, "Stringer")
	reflectx.SetInterfaceType(typ, nil,
		[]reflect.Method{
			reflect.Method{
				Name: "String",
				Type: reflect.FuncOf(nil, []reflect.Type{tyString}, false),
			},
		},
	)
	checkInterface(t, typ, reflect.TypeOf((*fmt.Stringer)(nil)).Elem())

	typ = reflectx.NewInterfaceType(pkgpath, "ReadWriteCloser")
	reflectx.SetInterfaceType(typ,
		[]reflect.Type{
			reflect.TypeOf((*io.Reader)(nil)).Elem(),
			reflect.TypeOf((*io.Writer)(nil)).Elem(),
		},
		[]reflect.Method{
			reflect.Method{
				Name: "Close",
				Type: reflect.FuncOf(nil, []reflect.Type{tyError}, false),
			},
		},
	)
	checkInterface(t, typ, reflect.TypeOf((*io.ReadWriteCloser)(nil)).Elem())
}

type MyPoint1 struct {
	Point
}

type MyPoint2 struct {
	*Point
}

type Setter interface {
	Set(x int, y int)
	String() string
}

type MyPoint3 struct {
	Setter
}

type MyPoint4 struct {
	*Point
	index int
}

func (s *MyPoint4) SetIndex(n int) {
	s.index = n
}

func (s MyPoint4) Index() int {
	return s.index
}

func (s MyPoint4) String() string {
	return fmt.Sprintf("%v#%v", s.index, s.Point)
}

func makeDynamicSetterType() reflect.Type {
	return reflectx.NamedInterfaceOf("main", "Setter", nil,
		[]reflect.Method{
			reflect.Method{
				Name: "Set",
				Type: reflect.FuncOf([]reflect.Type{tyInt, tyInt}, nil, false),
			},
			reflect.Method{
				Name: "String",
				Type: reflect.FuncOf(nil, []reflect.Type{tyString}, false),
			},
		},
	)
}

func TestEmbedMethods1(t *testing.T) {
	// MyPoint1
	typ := reflect.TypeOf((*MyPoint1)(nil)).Elem()
	if v := typ.NumMethod(); v != 4 {
		t.Fatalf("NumMethod have %v want 4", v)
	}
	if v := reflect.PtrTo(typ).NumMethod(); v != 5 {
		t.Fatalf("NumMethod have %v want 5", v)
	}
	fnTest := func(t *testing.T, tyPoint reflect.Type) {
		fs := []reflect.StructField{
			reflect.StructField{
				Name:      "Point",
				Type:      tyPoint,
				Anonymous: true,
			},
		}
		typ := reflectx.NamedStructOf("main", "MyPoint1", fs)
		typ = reflectx.StructToMethodSet(typ)
		if v := typ.NumMethod(); v != 4 {
			t.Errorf("NumMethod have %v want 4", v)
		}
		if v := reflect.PtrTo(typ).NumMethod(); v != 5 {
			t.Errorf("NumMethod have %v want 5", v)
		}
		m := reflect.New(typ).Elem()
		m.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf(200)})
		if v := fmt.Sprint(m); v != "(100,200)" {
			t.Errorf("have %v want (100,200)", v)
		}
		if v := fmt.Sprint(m.Addr()); v != "(100,200)" {
			t.Errorf("have %v want (100,200)", v)
		}
		m.Field(0).Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(-100), reflect.ValueOf(-200)})
		if v := fmt.Sprint(m.Field(0)); v != "(-100,-200)" {
			t.Errorf("have %v want (-100,-200)", v)
		}
		if v := fmt.Sprint(m.Field(0).Addr()); v != "(-100,-200)" {
			t.Errorf("have %v want (-100,-200)", v)
		}
	}

	// test mixed embed struct
	fnTest(t, reflect.TypeOf((*Point)(nil)).Elem())
	// test dynamic embed struct
	fnTest(t, makeDynamicPointType())
}

func TestEmbedMethods2(t *testing.T) {
	// MyPoint2
	typ := reflect.TypeOf((*MyPoint2)(nil)).Elem()
	if v := typ.NumMethod(); v != 5 {
		t.Fatalf("NumMethod have %v want 5", v)
	}
	if v := reflect.PtrTo(typ).NumMethod(); v != 5 {
		t.Fatalf("NumMethod have %v want 5", v)
	}

	// embbed ptr
	fnTest := func(t *testing.T, tyPoint reflect.Type) {
		fs := []reflect.StructField{
			reflect.StructField{
				Name:      "Point",
				Type:      reflect.PtrTo(tyPoint),
				Anonymous: true,
			},
		}
		typ = reflectx.NamedStructOf("main", "MyPoint2", fs)
		typ = reflectx.StructToMethodSet(typ)
		if v := typ.NumMethod(); v != 5 {
			t.Errorf("NumMethod have %v want 5", v)
		}
		if v := reflect.PtrTo(typ).NumMethod(); v != 5 {
			t.Errorf("NumMethod have %v want 5", v)
		}
		m := reflect.New(typ).Elem()
		m.Field(0).Set(reflect.New(tyPoint))
		m.MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf(200)})
		if v := fmt.Sprint((m)); v != "(100,200)" {
			t.Errorf("have %v want (100,200)", v)
		}
		if v := fmt.Sprint(m.Addr()); v != "(100,200)" {
			t.Errorf("have %v want (100,200)", v)
		}
		m.Field(0).MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(-100), reflect.ValueOf(-200)})
		if v := fmt.Sprint(m); v != "(-100,-200)" {
			t.Errorf("have %v want (-100,-200)", v)
		}
		if v := fmt.Sprint(m.Field(0)); v != "(-100,-200)" {
			t.Errorf("have %v want (-100,-200)", v)
		}
		if v := fmt.Sprint(m.Field(0).Elem()); v != "(-100,-200)" {
			t.Errorf("have %v want (-100,-200)", v)
		}
		m.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(300), reflect.ValueOf(400)})
		if v := fmt.Sprint(m); v != "(300,400)" {
			t.Errorf("have %v want (300,400)", v)
		}
		if v := fmt.Sprint((m.Addr())); v != "(300,400)" {
			t.Errorf("have %v want (300,400)", v)
		}
	}
	// test mixed embed ptr
	fnTest(t, reflect.TypeOf((*Point)(nil)).Elem())
	// test dynamic embed ptr
	fnTest(t, makeDynamicPointType())
}

func TestEmbedMethods3(t *testing.T) {
	// MyPoint3
	typ := reflect.TypeOf((*MyPoint3)(nil)).Elem()
	if v := typ.NumMethod(); v != 2 {
		t.Fatalf("NumMethod have %v want 2", v)
	}
	if v := reflect.PtrTo(typ).NumMethod(); v != 2 {
		t.Fatalf("NumMethod have %v want 2", v)
	}
	var i MyPoint3
	i.Setter = &Point{}
	i.Set(100, 200)
	if v := fmt.Sprint(i); v != "(100,200)" {
		t.Fatalf("String have %v, want (100,200)", v)
	}
	(&i).Set(300, 400)
	if v := fmt.Sprint(i); v != "(300,400)" {
		t.Fatalf("String have %v, want (300,400)", v)
	}

	// embbed interface
	fnTest := func(t *testing.T, setter reflect.Type, tyPoint reflect.Type) {
		fs := []reflect.StructField{
			reflect.StructField{
				Name:      "Setter",
				Type:      setter,
				Anonymous: true,
			},
		}
		typ := reflectx.NamedStructOf("main", "MyPoint3", fs)
		typ = reflectx.StructToMethodSet(typ)
		if v := typ.NumMethod(); v != 2 {
			t.Errorf("NumMethod have %v want 2", v)
		}
		if v := reflect.PtrTo(typ).NumMethod(); v != 2 {
			t.Errorf("NumMethod have %v want 2", v)
		}
		m := reflect.New(typ).Elem()
		m.Field(0).Set(reflect.New(tyPoint))
		m.MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf(200)})
		if v := fmt.Sprint((m)); v != "(100,200)" {
			t.Errorf("have %v want (100,200)", v)
		}
		m.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(300), reflect.ValueOf(400)})
		if v := fmt.Sprint((m)); v != "(300,400)" {
			t.Errorf("have %v want (300,400)", v)
		}
	}
	// test mixed embed interface
	fnTest(t, reflect.TypeOf((*Setter)(nil)).Elem(), reflect.TypeOf((*Point)(nil)).Elem())
	fnTest(t, reflect.TypeOf((*Setter)(nil)).Elem(), makeDynamicPointType())
	// test dynamic embed interface
	fnTest(t, makeDynamicSetterType(), reflect.TypeOf((*Point)(nil)).Elem())
	fnTest(t, makeDynamicSetterType(), makeDynamicPointType())
}

func TestEmbedMethods4(t *testing.T) {
	// MyPoint4
	typ := reflect.TypeOf((*MyPoint4)(nil)).Elem()
	if v := typ.NumMethod(); v != 6 {
		t.Fatalf("NumMethod have %v want 6", v)
	}
	if v := reflect.PtrTo(typ).NumMethod(); v != 7 {
		t.Fatalf("NumMethod have %v want 7", v)
	}
	var i MyPoint4
	i.Point = &Point{}
	i.Set(100, 200)
	if v := fmt.Sprint(i); v != "0#(100,200)" {
		t.Fatalf("String have %v, want 0#(100,200)", v)
	}
	i.SetIndex(1)
	i.Set(300, 400)
	if v := fmt.Sprint(i); v != "1#(300,400)" {
		t.Fatalf("String have %v, want 1#(300,400)", v)
	}

	fnTest := func(t *testing.T, tyPoint reflect.Type) {
		// embbed ptr
		fs := []reflect.StructField{
			reflect.StructField{
				Name:      "Point",
				Type:      reflect.PtrTo(tyPoint),
				Anonymous: true,
			},
			reflect.StructField{
				Name:      "index",
				PkgPath:   "main",
				Type:      reflect.TypeOf(int(0)),
				Anonymous: false,
			},
		}
		mSetIndex := reflectx.MakeMethod(
			"SetIndex",
			"main",
			true,
			reflect.FuncOf([]reflect.Type{tyInt}, nil, false),
			func(args []reflect.Value) []reflect.Value {
				reflectx.Field(args[0].Elem(), 1).SetInt(args[1].Int())
				return nil
			},
		)
		mIndex := reflectx.MakeMethod(
			"Index",
			"main",
			false,
			reflect.FuncOf(nil, []reflect.Type{tyInt}, false),
			func(args []reflect.Value) []reflect.Value {
				return []reflect.Value{args[0].Field(1)}
			},
		)
		mString := reflectx.MakeMethod(
			"String",
			"main",
			false,
			reflect.FuncOf(nil, []reflect.Type{tyString}, false),
			func(args []reflect.Value) []reflect.Value {
				info := fmt.Sprintf("%v#%v", args[0].Field(1), args[0].Field(0))
				return []reflect.Value{reflect.ValueOf(info)}
			},
		)
		typ := reflectx.NamedStructOf("main", "MyPoint4", fs)
		typ = reflectx.NewMethodSet(typ, 2, 3)
		reflectx.SetMethodSet(typ, []reflectx.Method{
			mSetIndex,
			mIndex,
			mString,
		}, true)
		if v := typ.NumMethod(); v != 6 {
			t.Errorf("NumMethod have %v want 6", v)
		}
		if v := reflect.PtrTo(typ).NumMethod(); v != 7 {
			t.Errorf("NumMethod have %v want 7", v)
		}
		m := reflect.New(typ).Elem()
		m.Field(0).Set(reflect.New(tyPoint))
		m.MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf(200)})
		if v := fmt.Sprint(m); v != "0#(100,200)" {
			t.Errorf("have %v want 0#(100,200)", v)
		}
		m.Addr().MethodByName("SetIndex").Call([]reflect.Value{reflect.ValueOf(1)})
		m.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(300), reflect.ValueOf(400)})
		if v := fmt.Sprint(m); v != "1#(300,400)" {
			t.Errorf("have %v want 1#(300,400)", v)
		}
	}

	// test mixed embed ptr with methods
	fnTest(t, reflect.TypeOf((*Point)(nil)).Elem())
	// test dynamic embed ptr with methods
	fnTest(t, makeDynamicPointType())
}

type itoaFunc func(i int) string

func (f itoaFunc) Itoa(i int) string { return f(i) }

type Itoa interface {
	Itoa(i int) string
}

func TestFunc(t *testing.T) {
	if runtime.Compiler == "gopherjs" {
		t.Skip("skip gopherjs")
	}
	fn := itoaFunc(func(i int) string {
		return strconv.Itoa(i)
	})
	if fn(100) != "100" {
		t.Fail()
	}
	if Itoa(fn).Itoa(100) != "100" {
		t.Fail()
	}
	fnTyp := reflect.TypeOf((*itoaFunc)(nil)).Elem()
	fnValue := reflect.MakeFunc(fnTyp, func(args []reflect.Value) []reflect.Value {
		r := strconv.Itoa(int(args[0].Int()))
		return []reflect.Value{reflect.ValueOf(r)}
	})
	if i := fnValue.Interface().(Itoa); i.Itoa(100) != "100" {
		t.Fail()
	}
	styp := reflectx.NamedTypeOf("main", "itoaFunc", reflect.FuncOf([]reflect.Type{tyInt}, []reflect.Type{tyString}, false))
	typ := reflectx.NewMethodSet(styp, 1, 1)
	mItoa := reflectx.MakeMethod("Itoa", "main", false,
		reflect.FuncOf([]reflect.Type{tyInt}, []reflect.Type{tyString}, false),
		func(args []reflect.Value) []reflect.Value {
			return args[0].Call(args[1:])
		})
	err := reflectx.SetMethodSet(typ, []reflectx.Method{mItoa}, false)
	if err != nil {
		t.Errorf("SetMethodSet error: %v", err)
	}
	if typ.NumMethod() != 1 {
		t.Fail()
	}
	v := reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
		r := strconv.Itoa(int(args[0].Int()))
		return []reflect.Value{reflect.ValueOf(r)}
	})
	if v.NumMethod() != 1 {
		t.Fail()
	}
	if r := v.Call([]reflect.Value{reflect.ValueOf(100)}); r[0].String() != "100" {
		t.Fail()
	}
	if r := reflectx.MethodByIndex(typ, 0).Func.Call([]reflect.Value{v, reflect.ValueOf(100)}); r[0].String() != "100" {
		t.Fail()
	}
	if r := v.Method(0).Call([]reflect.Value{reflect.ValueOf(100)}); r[0].String() != "100" {
		t.Fail()
	}
}

type chanType chan int

func (ch chanType) Send(n int) {
	ch <- n
}

func (ch chanType) Recv() (n int) {
	t := time.NewTimer(1e9)
	defer t.Stop()
	select {
	case n = <-ch:
	case <-t.C:
		n = -1
	}
	return
}

func TestChan(t *testing.T) {
	if runtime.Compiler == "gopherjs" {
		t.Skip("skip gopherjs")
	}
	c := make(chanType)
	go func() {
		c.Send(100)
	}()
	if n := c.Recv(); n != 100 {
		t.Fatalf("recv %v", n)
	}
	styp := reflectx.NamedTypeOf("main", "chanType", reflect.TypeOf((chan int)(nil)))
	typ := reflectx.NewMethodSet(styp, 2, 2)
	mSend := reflectx.MakeMethod(
		"Send",
		"main",
		false,
		reflect.FuncOf([]reflect.Type{tyInt}, nil, false),
		func(args []reflect.Value) []reflect.Value {
			args[0].Send(args[1])
			return nil
		})
	mRecv := reflectx.MakeMethod(
		"Recv",
		"main",
		false,
		reflect.FuncOf(nil, []reflect.Type{tyInt}, false),
		func(args []reflect.Value) []reflect.Value {
			t := time.NewTimer(1e9)
			n, r, _ := reflect.Select([]reflect.SelectCase{
				{reflect.SelectRecv, args[0], reflect.Value{}},
				{reflect.SelectRecv, reflect.ValueOf(t.C), reflect.Value{}},
			})
			if n != 0 {
				return []reflect.Value{reflect.ValueOf(-1)}
			}
			return []reflect.Value{r}
		})
	err := reflectx.SetMethodSet(typ, []reflectx.Method{
		mSend,
		mRecv,
	}, false)
	if err != nil {
		t.Errorf("SetMethodSet error: %v", err)
	}
	if typ.NumMethod() != 2 {
		t.Fatal()
	}
	ch := reflect.MakeChan(typ, 0)
	go func() {
		ch.MethodByName("Send").Call([]reflect.Value{reflect.ValueOf(100)})
	}()
	if r := ch.MethodByName("Recv").Call(nil); r[0].Int() != 100 {
		t.Fatalf("recv %v", r[0])
	}
}

type Map map[int]string

func (m Map) Set(k int, v string) {
	m[k] = v
}

func (m Map) Get(k int) (string, bool) {
	r, ok := m[k]
	return r, ok
}

func TestMap(t *testing.T) {
	{
		m := make(Map)
		m.Set(100, "Hello")
		r, ok := m.Get(100)
		if !ok {
			t.Fail()
		}
		if r != "Hello" {
			t.Fail()
		}
	}
	styp := reflectx.NamedTypeOf("main", "Map", reflect.TypeOf((*map[int]string)(nil)).Elem())
	typ := reflectx.NewMethodSet(styp, 2, 2)
	mSet := reflectx.MakeMethod(
		"Set",
		"main",
		false,
		reflect.FuncOf([]reflect.Type{tyInt, tyString}, nil, false),
		func(args []reflect.Value) []reflect.Value {
			args[0].SetMapIndex(args[1], args[2])
			return nil
		})
	mGet := reflectx.MakeMethod(
		"Get",
		"main",
		false,
		reflect.FuncOf([]reflect.Type{tyInt}, []reflect.Type{tyString, tyBool}, false),
		func(args []reflect.Value) []reflect.Value {
			r := args[0].MapIndex(args[1])
			if r.IsValid() {
				return []reflect.Value{r, reflect.ValueOf(true)}
			}
			return []reflect.Value{r, reflect.ValueOf(false)}
		})
	err := reflectx.SetMethodSet(typ, []reflectx.Method{
		mSet,
		mGet,
	}, false)
	if err != nil {
		t.Errorf("SetMethodSet error: %v", err)
	}
	if typ.NumMethod() != 2 {
		t.Fatal()
	}
	v := reflect.MakeMap(typ)
	v.MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf("Hello")})
	r := v.MethodByName("Get").Call([]reflect.Value{reflect.ValueOf(100)})
	if len(r) != 2 {
		t.Fail()
	}
	if fmt.Sprint(r[0]) != "Hello" || fmt.Sprint(r[1]) != "true" {
		t.Fatal(r[0], r[1])
	}
}

type emtpyCall struct {
	X int
	Y int
}

//go:noinline
func (t *emtpyCall) Set(x int, y int) {

}

//go:noinline
func (t emtpyCall) Info(x int, y int) {

}

func makeDynamicEmptyCall() reflect.Type {
	fs := []reflect.StructField{
		reflect.StructField{Name: "X", Type: reflect.TypeOf(0)},
		reflect.StructField{Name: "Y", Type: reflect.TypeOf(0)},
	}
	styp := reflectx.NamedStructOf("main", "emptyCall", fs)
	//var typ reflect.Type
	typ := reflectx.NewMethodSet(styp, 1, 2)
	mInfo := reflectx.MakeMethod(
		"Info",
		"main",
		false,
		reflect.FuncOf([]reflect.Type{tyInt, tyInt}, nil, false),
		func(args []reflect.Value) (result []reflect.Value) {
			return
		},
	)
	mSet := reflectx.MakeMethod(
		"Set",
		"main",
		true,
		reflect.FuncOf([]reflect.Type{tyInt, tyInt}, nil, false),
		func(args []reflect.Value) (result []reflect.Value) {
			return
		},
	)
	reflectx.SetMethodSet(typ, []reflectx.Method{
		mInfo,
		mSet,
	}, true)
	return typ
}

func BenchmarkNativeCallPtr(b *testing.B) {
	pt := &emtpyCall{}
	for i := 0; i < b.N; i++ {
		pt.Set(100, 200)
	}
}

func BenchmarkReflectCallPtr(b *testing.B) {
	b.StopTimer()
	pt := &emtpyCall{}
	set := reflect.ValueOf(pt).MethodByName("Set").Interface().(func(int, int))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		set(100, 200)
	}
}

func BenchmarkDynamicCallPtr(b *testing.B) {
	b.StopTimer()
	typ := makeDynamicEmptyCall()
	pt := reflect.New(typ)
	set := pt.MethodByName("Set").Interface().(func(int, int))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		set(100, 200)
	}
}

func BenchmarkNativeCallNoPtr(b *testing.B) {
	pt := emtpyCall{}
	for i := 0; i < b.N; i++ {
		pt.Info(100, 200)
	}
}

func BenchmarkReflectCallNoPtr(b *testing.B) {
	b.StopTimer()
	pt := emtpyCall{}
	set := reflect.ValueOf(pt).MethodByName("Info").Interface().(func(int, int))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		set(100, 200)
	}
}

func BenchmarkDynamicCallNoPtr(b *testing.B) {
	b.StopTimer()
	typ := makeDynamicEmptyCall()
	pt := reflect.New(typ).Elem()
	set := pt.MethodByName("Info").Interface().(func(int, int))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		set(100, 200)
	}
}

func BenchmarkNativeCallIndirect(b *testing.B) {
	pt := &emtpyCall{}
	for i := 0; i < b.N; i++ {
		pt.Info(100, 200)
	}
}

func BenchmarkReflectCallIndirect(b *testing.B) {
	b.StopTimer()
	pt := &emtpyCall{}
	set := reflect.ValueOf(pt).MethodByName("Info").Interface().(func(int, int))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		set(100, 200)
	}
}

func BenchmarkDynamicCallIndirect(b *testing.B) {
	b.StopTimer()
	typ := makeDynamicEmptyCall()
	pt := reflect.New(typ)
	set := pt.MethodByName("Info").Interface().(func(int, int))
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		set(100, 200)
	}
}
