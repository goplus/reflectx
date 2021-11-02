package reflectx_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	"github.com/goplus/reflectx"
)

type nPoint struct {
	x int
	y int
}

func TestFieldCanSet(t *testing.T) {
	x := &nPoint{10, 20}
	v := reflect.ValueOf(x).Elem()

	sf := v.Field(0)
	if sf.CanSet() {
		t.Fatal("x unexport cannot set")
	}

	sf = reflectx.CanSet(sf)
	if !sf.CanSet() {
		t.Fatal("CanSet failed")
	}

	sf.Set(reflect.ValueOf(201))
	if x.x != 201 {
		t.Fatalf("x value %v", x.x)
	}
	sf.SetInt(202)
	if x.x != 202 {
		t.Fatalf("x value %v", x.x)
	}
}

type Rect struct {
	pt1 nPoint
	pt2 *nPoint
}

func TestField(t *testing.T) {
	x := &Rect{nPoint{1, 2}, &nPoint{3, 4}}
	v := reflect.ValueOf(x).Elem()
	reflectx.Field(v, 0).Set(reflect.ValueOf(nPoint{10, 20}))
	if x.pt1.x != 10 || x.pt1.y != 20 {
		t.Fatalf("pt1 %v", x.pt1)
	}
	reflectx.FieldByName(v, "pt2").Set(reflect.ValueOf(&nPoint{30, 40}))
	if x.pt2.x != 30 || x.pt2.y != 40 {
		t.Fatalf("pt2 %v", x.pt2)
	}
	reflectx.FieldByNameFunc(v, func(name string) bool {
		return name == "pt2"
	}).Set(reflect.ValueOf(&nPoint{50, 60}))
	if x.pt2.x != 50 || x.pt2.y != 60 {
		t.Fatalf("pt2 %v", x.pt2)
	}
	reflectx.FieldByIndex(v, []int{0, 1}).SetInt(100)
	if x.pt1.y != 100 {
		t.Fatalf("pt1.y %v", x.pt1)
	}
}

func TestFieldX(t *testing.T) {
	x := &Rect{nPoint{1, 2}, &nPoint{3, 4}}
	v := reflect.ValueOf(x).Elem()
	reflectx.FieldX(v, 0).Set(reflect.ValueOf(nPoint{10, 20}))
	if x.pt1.x != 10 || x.pt1.y != 20 {
		t.Fatalf("pt1 %v", x.pt1)
	}
	reflectx.FieldByNameX(v, "pt2").Set(reflect.ValueOf(&nPoint{30, 40}))
	if x.pt2.x != 30 || x.pt2.y != 40 {
		t.Fatalf("pt2 %v", x.pt2)
	}
	reflectx.FieldByNameFuncX(v, func(name string) bool {
		return name == "pt2"
	}).Set(reflect.ValueOf(&nPoint{50, 60}))
	if x.pt2.x != 50 || x.pt2.y != 60 {
		t.Fatalf("pt2 %v", x.pt2)
	}
	reflectx.FieldByIndexX(v, []int{0, 1}).SetInt(100)
	if x.pt1.y != 100 {
		t.Fatalf("pt1.y %v", x.pt1)
	}
}

func TestStructOfUnderscore(t *testing.T) {
	fs := []reflect.StructField{
		reflect.StructField{
			Name:    "_",
			PkgPath: "main",
			Type:    tyInt,
		},
		reflect.StructField{
			Name:    "_",
			PkgPath: "main",
			Type:    tyInt,
		},
	}
	typ := reflectx.NamedStructOf("main", "Point", fs)
	if typ.Field(0).Name != "_" {
		t.Fatalf("field name must underscore")
	}
	if typ.Field(1).Name != "_" {
		t.Fatalf("field name must underscore")
	}
}

func TestStructOfExport(t *testing.T) {
	fs := []reflect.StructField{
		reflect.StructField{
			Name:    "x",
			PkgPath: "main",
			Type:    tyInt,
		},
		reflect.StructField{
			Name:    "y",
			PkgPath: "main",
			Type:    tyInt,
		},
	}
	typ := reflectx.NamedStructOf("main", "Point", fs)
	v := reflect.New(typ).Elem()
	reflectx.FieldByIndex(v, []int{0}).SetInt(100)
	reflectx.FieldByIndex(v, []int{1}).SetInt(200)
	if s := fmt.Sprint(v); s != "{100 200}" {
		t.Fatalf("have %v, want {100 200}", s)
	}
}

type Buffer struct {
	*bytes.Buffer
	size  int
	value reflect.Value
	*bytes.Reader
}

func TestStructOf(t *testing.T) {
	defer func() {
		v := recover()
		if v != nil {
			t.Fatalf("reflectx.StructOf %v", v)
		}
	}()
	typ := reflect.TypeOf((*Buffer)(nil)).Elem()
	var fs []reflect.StructField
	for i := 0; i < typ.NumField(); i++ {
		fs = append(fs, typ.Field(i))
	}
	dst := reflectx.StructOf(fs)
	for i := 0; i < dst.NumField(); i++ {
		if dst.Field(i).Anonymous != fs[i].Anonymous {
			t.Errorf("error field %v", dst.Field(i))
		}
	}

	v := reflect.New(dst)
	v.Elem().Field(0).Set(reflect.ValueOf(bytes.NewBufferString("hello")))
	reflectx.CanSet(v.Elem().Field(1)).SetInt(100)
}

func TestNamedStruct(t *testing.T) {
	fs := []reflect.StructField{
		reflect.StructField{Name: "X", Type: reflect.TypeOf(0)},
		reflect.StructField{Name: "Y", Type: reflect.TypeOf(0)},
	}
	t1 := reflect.StructOf(fs)
	t2 := reflect.StructOf(fs)
	if t1 != t2 {
		t.Fatalf("reflect.StructOf %v != %v", t1, t2)
	}
	t3 := reflectx.NamedStructOf("github.com/goplus/reflectx_test", "Point", fs)
	t4 := reflectx.NamedStructOf("github.com/goplus/reflectx_test", "Point2", fs)
	if t3 == t4 {
		t.Fatalf("NamedStructOf %v == %v", t3, t4)
	}
	if t4.String() != "reflectx_test.Point2" {
		t.Fatalf("t4.String=%v", t4.String())
	}
	if t4.Name() != "Point2" {
		t.Fatalf("t4.Name=%v", t4.Name())
	}
	if t4.PkgPath() != "github.com/goplus/reflectx_test" {
		t.Fatalf("t4.PkgPath=%v", t4.PkgPath())
	}
}

var (
	ch = make(chan bool)
	fn = func(int, string) (bool, int) {
		return true, 0
	}
	fn2 = func(*nPoint, int, bool, []byte) int {
		return 0
	}
	testNamedValue = []interface{}{
		true,
		false,
		int(2),
		int8(3),
		int16(4),
		int32(5),
		int64(6),
		uint(7),
		uint8(8),
		uint16(9),
		uint32(10),
		uint64(11),
		uintptr(12),
		float32(13),
		float64(14),
		complex64(15),
		complex128(16),
		"hello",
		unsafe.Pointer(nil),
		unsafe.Pointer(&fn),
		[]byte("hello"),
		[]int{1, 2, 3},
		[5]byte{'a', 'b', 'c', 'd', 'e'},
		[5]int{1, 2, 3, 4, 5},
		[]string{"a", "b"},
		[]int{100, 200},
		map[int]string{1: "hello", 2: "world"},
		new(uint8),
		&fn,
		&fn2,
		&ch,
		ch,
		fn,
		fn2,
	}
)

func TestNamedType(t *testing.T) {
	pkgpath := "github.com/goplus/reflectx"
	for i, v := range testNamedValue {
		value := reflect.ValueOf(v)
		typ := value.Type()
		nt := reflectx.NamedTypeOf("github.com/goplus/reflectx", fmt.Sprintf("MyType%v", i), typ)
		if nt.Kind() != typ.Kind() {
			t.Errorf("kind: have %v, want %v", nt.Kind(), typ.Kind())
		}
		if nt == typ {
			t.Errorf("same type, %v", typ)
		}
		name := fmt.Sprintf("My_Type%v", i)
		nt2 := reflectx.NamedTypeOf(pkgpath, name, typ)
		if nt == nt2 {
			t.Errorf("same type, %v", nt)
		}
		nv := reflect.New(nt).Elem()
		reflectx.SetValue(nv, value) //
		s1 := fmt.Sprint((nv))
		s2 := fmt.Sprint(v)
		if s1 != s2 {
			t.Errorf("%v: have %v, want %v", nt.Kind(), s1, s2)
		}
		if nt2.Name() != name {
			t.Errorf("name: have %v, want %v", nt2.Name(), name)
		}
		if nt2.PkgPath() != pkgpath {
			t.Errorf("pkgpath: have %v, want %v", nt2.PkgPath(), pkgpath)
		}
	}
}

var testInterfaceType = []reflect.Type{
	reflect.TypeOf((*interface{})(nil)).Elem(),
	reflect.TypeOf((*fmt.Stringer)(nil)).Elem(),
	reflect.TypeOf((*interface {
		Read(p []byte) (n int, err error)
		Write(p []byte) (n int, err error)
		Close() error
	})(nil)),
}

func TestNamedInterface(t *testing.T) {
	pkgpath := reflect.TypeOf((*interface{})(nil)).Elem().PkgPath()
	for i, styp := range testInterfaceType {
		name := fmt.Sprintf("T%v", i)
		typ := reflectx.NamedTypeOf(pkgpath, name, styp)
		if typ.Name() != name {
			t.Errorf("name: have %v, want %v", typ.Name(), name)
		}
		if typ.PkgPath() != pkgpath {
			t.Errorf("pkgpath: have %v, want %v", typ.PkgPath(), pkgpath)
		}
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
}

func TestNamedTypeStruct(t *testing.T) {
	typ := reflect.TypeOf((*nPoint)(nil)).Elem()
	pkgpath := typ.PkgPath()
	nt := reflectx.NamedTypeOf(pkgpath, "MyPoint", typ)
	nt2 := reflectx.NamedTypeOf(pkgpath, "MyPoint2", typ)
	if nt.NumField() != typ.NumField() {
		t.Fatal("NumField != 2", nt.NumField())
	}
	if nt.Name() != "MyPoint" {
		t.Fatal("Name != MyPoint", nt.Name())
	}
	if nt == nt2 {
		t.Fatalf("same type %v", nt)
	}
	v := reflect.New(nt).Elem()
	reflectx.Field(v, 0).SetInt(100)
	reflectx.Field(v, 1).SetInt(200)
	if v.FieldByName("x").Int() != 100 || v.FieldByName("y").Int() != 200 {
		t.Fatal("Value != {100 200},", v)
	}
}

func TestSetElem(t *testing.T) {
	typ := reflectx.NamedTypeOf("main", "T", reflect.TypeOf(([]struct{})(nil)))
	reflectx.SetElem(typ, typ)
	v := reflect.MakeSlice(typ, 3, 3)
	v.Index(0).Set(reflect.MakeSlice(typ, 1, 1))
	v.Index(1).Set(reflect.MakeSlice(typ, 2, 2))
	s := fmt.Sprintf("%v", v.Interface())
	if s != "[[[]] [[] []] []]" {
		t.Fatalf("failed SetElem s=%v", s)
	}
}
