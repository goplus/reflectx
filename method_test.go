package reflectx_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/goplus/reflectx"
)

var (
	byteTyp = reflect.TypeOf(byte('a'))
	boolTyp = reflect.TypeOf(true)
	intTyp  = reflect.TypeOf(0)
	strTyp  = reflect.TypeOf("")
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
	styp := reflectx.NamedTypeOf("main", "Int", intTyp)
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

	x := reflectx.New(typ).Elem()
	x.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100)})

	// String
	if v := fmt.Sprint(reflectx.Interface(x)); v != "(100)" {
		t.Fatalf("String(): have %v, want (100)", v)
	}
	if v := fmt.Sprint(reflectx.Interface(x.Addr())); v != "(100)" {
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
	var typ reflect.Type
	mString := reflectx.MakeMethod(
		"String",
		false,
		reflect.FuncOf(nil, []reflect.Type{strTyp}, false),
		func(args []reflect.Value) []reflect.Value {
			v := args[0]
			info := fmt.Sprintf("{%v}%v", v.Len(), v.Convert(intSliceTyp))
			return []reflect.Value{reflect.ValueOf(info)}
		},
	)
	mSet := reflectx.MakeMethod(
		"Set",
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

	x := reflectx.New(typ).Elem()
	x.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf(200), reflect.ValueOf(300)})

	// String
	if v := fmt.Sprint(reflectx.Interface(x)); v != "{3}[100 200 300]" {
		t.Fatalf("String(): have %v, want {3}[100 200 300]", v)
	}
	if v := fmt.Sprint(reflectx.Interface(x.Addr())); v != "{3}[100 200 300]" {
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
	// make IntArray
	styp := reflectx.NamedTypeOf("main", "IntArray", reflect.TypeOf([2]int{}))
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
	mScale := reflectx.MakeMethod(
		"Scale",
		false,
		reflect.FuncOf([]reflect.Type{intTyp}, []reflect.Type{styp}, false),
		func(args []reflect.Value) (result []reflect.Value) {
			v := args[0]
			s := args[1].Int()
			r := reflect.New(typ).Elem()
			r.Index(0).SetInt(v.Index(0).Int() * s)
			r.Index(1).SetInt(v.Index(1).Int() * s)
			return []reflect.Value{r}
		},
	)
	typ = reflectx.MethodOf(styp, []reflectx.Method{
		mString,
		mSet,
		mGet,
		mScale,
	})
	ptrType := reflect.PtrTo(typ)

	if n := typ.NumMethod(); n != 3 {
		t.Fatal("typ.NumMethod()", n)
	}
	if n := ptrType.NumMethod(); n != 4 {
		t.Fatal("ptrTyp.NumMethod()", n)
	}

	x := reflectx.New(typ).Elem()
	x.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf(200)})

	// String
	if v := fmt.Sprint(reflectx.Interface(x)); v != "(100,200)" {
		t.Fatalf("String(): have %v, want (100,200)", v)
	}
	if v := fmt.Sprint(reflectx.Interface(x.Addr())); v != "(100,200)" {
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
	if v := fmt.Sprint(reflectx.Interface(r[0])); v != "(500,1000)" {
		t.Fatalf("typ reflectx.MethodByName Scale: have %v, want (500,1000)", v)
	}
	r = x.MethodByName("Scale").Call([]reflect.Value{reflect.ValueOf(5)})
	if v := fmt.Sprint(reflectx.Interface(r[0])); v != "(500,1000)" {
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

func TestStructMethodOf(t *testing.T) {
	// Point
	var i Point
	i.Set(100, 200)
	if v := fmt.Sprint(i); v != "(100,200)" {
		t.Fatalf("want %v, have (100,200)", v)
	}
	if v := fmt.Sprint(i.Add(Point{1, 2})); v != "(101,202)" {
		t.Fatalf("want %v, have (101,202)", v)
	}
	if v := fmt.Sprint(i.Scale(2, 3, 4)); v != "[(200,400) (300,600) (400,800)]" {
		t.Fatalf("want %v, have [(200,400) (300,600) (400,800)]", v)
	}
	// make Point
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
	mScale := reflectx.MakeMethod(
		"Scale",
		false,
		reflect.FuncOf([]reflect.Type{reflect.SliceOf(intTyp)}, []reflect.Type{reflect.SliceOf(styp)}, true),
		func(args []reflect.Value) (result []reflect.Value) {
			x, y := args[0].Field(0).Int(), args[0].Field(1).Int()
			r := reflect.MakeSlice(reflect.SliceOf(typ), 0, 0)
			for i := 0; i < args[1].Len(); i++ {
				s := args[1].Index(i).Int()
				v := reflectx.New(typ).Elem()
				v.Field(0).SetInt(x * s)
				v.Field(1).SetInt(y * s)
				r = reflect.Append(r, v)
			}
			return []reflect.Value{r}
		},
	)
	typ = reflectx.MethodOf(styp, []reflectx.Method{
		mAdd,
		mString,
		mSet,
		mScale,
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
	r := m.Func.Call([]reflect.Value{pt1, pt2})
	if v := fmt.Sprint(reflectx.Interface(r[0])); v != "(400,600)" {
		t.Fatalf("type reflectx.MethodByName Add: have %v, want (400,600)", v)
	}
	r = pt1.MethodByName("Add").Call([]reflect.Value{pt2})
	if v := fmt.Sprint(reflectx.Interface(r[0])); v != "(400,600)" {
		t.Fatalf("value.MethodByName Add: have %v, want (400,600)", v)
	}

	// ptrtyp Add
	m, _ = reflectx.MethodByName(ptrType, "Add")
	r = m.Func.Call([]reflect.Value{pt1.Addr(), pt2})
	if v := fmt.Sprint(reflectx.Interface(r[0])); v != "(400,600)" {
		t.Fatalf("ptrType reflectx.MethodByName Add: have %v, want (400,600)", v)
	}
	r = pt1.Addr().MethodByName("Add").Call([]reflect.Value{pt2})
	if v := fmt.Sprint(reflectx.Interface(r[0])); v != "(400,600)" {
		t.Fatalf("ptrType value.reflectx.MethodByName Add: have %v, want (400,600)", v)
	}

	// Set
	m, _ = reflectx.MethodByName(ptrType, "Set")
	m.Func.Call([]reflect.Value{pt1.Addr(), reflect.ValueOf(-100), reflect.ValueOf(-200)})
	if v := fmt.Sprint(reflectx.Interface(pt1)); v != "(-100,-200)" {
		t.Fatalf("ptrType reflectx.MethodByName Set: have %v, want (-100,-200)", v)
	}
	pt1.Addr().MethodByName("Set").Call([]reflect.Value{reflect.ValueOf(100), reflect.ValueOf(200)})
	if v := fmt.Sprint(reflectx.Interface(pt1)); v != "(100,200)" {
		t.Fatalf("ptrType reflectx.MethodByName Set: have %v, want (100,200)", v)
	}

	// Scale
	m, _ = reflectx.MethodByName(typ, "Scale")
	r = m.Func.Call([]reflect.Value{pt1, reflect.ValueOf(2), reflect.ValueOf(3), reflect.ValueOf(4)})
	if v := fmt.Sprint(v2is(r[0])); v != "[(200,400) (300,600) (400,800)]" {
		t.Fatalf("want %v, have [(200,400) (300,600) (400,800)]", v)
	}
	r = pt1.MethodByName("Scale").Call([]reflect.Value{reflect.ValueOf(2), reflect.ValueOf(3), reflect.ValueOf(4)})
	if v := fmt.Sprint(v2is(r[0])); v != "[(200,400) (300,600) (400,800)]" {
		t.Fatalf("want %v, have [(200,400) (300,600) (400,800)]", v)
	}
}

func v2is(v reflect.Value) (is []interface{}) {
	for i := 0; i < v.Len(); i++ {
		is = append(is, reflectx.Interface(v.Index(i)))
	}
	return is
}
