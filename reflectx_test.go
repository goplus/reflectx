package reflectx_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/goplusjs/reflectx"
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

type MyPoint struct {
	pt1 Point
	pt2 *Point
}

func TestField(t *testing.T) {
	x := &MyPoint{Point{1, 2}, &Point{3, 4}}
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
			t.Failed()
		} else {
			t.Log("reflect.StructOf panic", v)
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
	t.Log(v.Interface())
}
