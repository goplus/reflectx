package reflectx_test

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/goplus/reflectx"
)

type Point struct {
	x int
	y int
}

func TestFieldCanSet(t *testing.T) {
	x := &Point{10, 20}
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
	pt1 Point
	pt2 *Point
}

func TestField(t *testing.T) {
	x := &Rect{Point{1, 2}, &Point{3, 4}}
	v := reflect.ValueOf(x).Elem()
	reflectx.Field(v, 0).Set(reflect.ValueOf(Point{10, 20}))
	if x.pt1.x != 10 || x.pt1.y != 20 {
		t.Fatalf("pt1 %v", x.pt1)
	}
	reflectx.FieldByName(v, "pt2").Set(reflect.ValueOf(&Point{30, 40}))
	if x.pt2.x != 30 || x.pt2.y != 40 {
		t.Fatalf("pt2 %v", x.pt2)
	}
	reflectx.FieldByNameFunc(v, func(name string) bool {
		return name == "pt2"
	}).Set(reflect.ValueOf(&Point{50, 60}))
	if x.pt2.x != 50 || x.pt2.y != 60 {
		t.Fatalf("pt2 %v", x.pt2)
	}
	reflectx.FieldByIndex(v, []int{0, 1}).SetInt(100)
	if x.pt1.y != 100 {
		t.Fatalf("pt1.y %v", x.pt1)
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
		if v == nil {
			t.Fatalf("reflect.StructOf panic")
		}
	}()
	typ := reflect.TypeOf((*Buffer)(nil)).Elem()
	var fs []reflect.StructField
	for i := 0; i < typ.NumField(); i++ {
		fs = append(fs, typ.Field(i))
	}
	reflect.StructOf(fs)
}

func TestStructOfX(t *testing.T) {
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
	testNamedType = []interface{}{
		true,
		false,
		uint8(1),
		uint16(2),
		uint32(3),
		uint64(4),
		int8(1),
		int16(2),
		int32(3),
		int64(4),
		float32(1.1),
		float64(1.2),
		100,
		1.23,
		"hello",
		[]byte("hello"),
		[5]byte{'a', 'b', 'c', 'd', 'e'},
		[]string{"a", "b"},
		[]int{100, 200},
		map[int]string{1: "hello", 2: "world"},
		new(uint8),
		ch,
		fn,
	}
)

func TestNamedType(t *testing.T) {
	for i, v := range testNamedType {
		value := reflect.ValueOf(v)
		typ := value.Type()
		nt := reflectx.NamedTypeOf("github.com/goplus/reflectx", fmt.Sprintf("MyType%v", i), typ)
		if nt.Kind() != typ.Kind() {
			t.Errorf("kind: have %v, want %v", nt.Kind(), typ.Kind())
		}
		if nt == typ {
			t.Errorf("same type, %v", typ)
		}
		nt2 := reflectx.NamedTypeOf("github.com/goplus/reflectx", fmt.Sprintf("My_Type%v", i), typ)
		if nt == nt2 {
			t.Errorf("same type, %v", nt)
		}
		nv := reflect.New(nt).Elem()
		reflectx.SetValue(nv, value)
		s1 := fmt.Sprintf("%v", nv)
		s2 := fmt.Sprintf("%v", v)
		if s1 != s2 {
			t.Errorf("%v: have %v, want %v", nt.Kind(), s1, s2)
		}
		t.Log(s1)
	}
}

func TestNamedTypeStruct(t *testing.T) {
	typ := reflect.TypeOf((*Point)(nil)).Elem()
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
