package reflectx_test

import (
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"testing"
	"unsafe"

	"github.com/goplus/reflectx"
)

var (
	tyInt = reflect.TypeOf(0)
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
			t.Errorf("kind: %v have %v, want %v", typ, nt.Kind(), typ.Kind())
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
			t.Errorf("pkgpath: %v have %v, want %v", typ, nt2.PkgPath(), pkgpath)
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
	pkgpath := "main"
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
	if runtime.Compiler == "gopherjs" {
		t.Skip("skip gopherjs")
	}
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

func TestNamedStructComparable(t *testing.T) {
	if runtime.Compiler == "gopherjs" {
		t.Skip("skip gopherjs")
	}
	fs := []reflect.StructField{
		reflect.StructField{Name: "_", PkgPath: "main", Type: reflect.TypeOf(0)},
		reflect.StructField{Name: "x", PkgPath: "main", Type: reflect.TypeOf(0)},
	}
	typ := reflectx.NamedStructOf("main", "blankStruct", fs)
	v1 := reflect.New(typ).Elem()
	reflectx.Field(v1, 0).SetInt(100)
	reflectx.Field(v1, 1).SetInt(200)
	v2 := reflect.New(typ).Elem()
	reflectx.Field(v2, 0).SetInt(-100)
	reflectx.Field(v2, 1).SetInt(200)
	if v1.Interface() != v2.Interface() {
		t.Fatal("failed struct equal")
	}
}

func TestNamedStructUncomparable(t *testing.T) {
	fs := []reflect.StructField{
		reflect.StructField{Name: "_", PkgPath: "main", Type: reflect.TypeOf(0)},
		reflect.StructField{Name: "fn", PkgPath: "main", Type: reflect.TypeOf(func() {})},
	}
	typ := reflectx.NamedStructOf("main", "funcStruct", fs)
	v1 := reflect.New(typ).Elem()
	v2 := reflect.New(typ).Elem()
	defer func() {
		if err := recover(); err == nil {
			t.Fatal("must panic comparing uncomparable type")
		}
	}()
	if v1.Interface() == v2.Interface() {
		t.Fatal("must panic")
	}
}

type point struct {
	x int
	y int
}

func (p *point) Set(x, y int) {
	p.x, p.y = x, y
}

func (p *point) mset(x, y int) {
	p.x, p.y = x, y
}

func (p *point) String() string {
	return fmt.Sprintf("(%v,%v)", p.x, p.y)
}

func TestMethodX(t *testing.T) {
	typ := reflect.TypeOf((*point)(nil))
	if n := reflectx.NumMethodX(typ); n != 3 {
		t.Fatalf("all method got: %v, want 3", n)
	}
	if m, ok := reflectx.MethodByName(typ, "mset"); !ok || m.Name != "mset" {
		t.Fatalf("failed lookup method mset %v\n", m)
	}
}

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

func TestSetUnderlyingStructFieldPkgPath(t *testing.T) {
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
	for _, source := range types {
		t.Run(source.name, func(t *testing.T) {
			dst := reflectx.NamedTypeOf("example.com/pkg", "T", reflect.TypeOf(struct{}{}))
			reflectx.SetUnderlying(dst, source.typ)
			for i := 0; i < source.typ.NumField(); i++ {
				want, got := source.typ.Field(i), dst.Field(i)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("field %d: got %+v, want %+v", i, got, want)
				}
			}
		})
	}
}

func TestSetUnderlyingInterfaceMethodPkgPath(t *testing.T) {
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
			dst := reflectx.NamedTypeOf("example.com/pkg", "I", reflect.TypeOf((*interface{})(nil)).Elem())
			reflectx.SetUnderlying(dst, source.typ)
			if dst.NumMethod() != source.typ.NumMethod() {
				t.Fatal("SetUnderlying changed the method count")
			}
			for i := 0; i < source.typ.NumMethod(); i++ {
				want, got := source.typ.Method(i), dst.Method(i)
				if got != want {
					t.Errorf("method %d: got %+v, want %+v", i, got, want)
				}
			}
			if !value.Type().Implements(dst) {
				t.Fatal("concrete value no longer implements the underlying interface")
			}
			v := reflect.New(dst).Elem()
			v.Set(value)
			if got := v.MethodByName("Public").Call(nil)[0].Int(); got != 7 {
				t.Fatalf("Public() = %d, want 7", got)
			}
		})
	}
}

func TestTypeLinksAndTypesByString(t *testing.T) {
	links := reflectx.TypeLinks()
	if len(links) == 0 {
		t.Fatal("TypeLinks empty")
	}
	_ = reflectx.TypesByString("int")
	_ = reflectx.TypesByString("this.type.does.not.exist.zzz")
	var buf bytes.Buffer
	reflectx.DumpType(&buf, reflect.TypeOf(nPoint{}))
	if buf.Len() == 0 {
		t.Fatal("DumpType empty")
	}
	reflectx.DumpType(&buf, reflect.TypeOf((*fmt.Stringer)(nil)).Elem())
}

func TestSetUnderlyingKinds(t *testing.T) {
	dstPtr := reflectx.NamedTypeOf("main", "P", reflect.TypeOf((*int)(nil)))
	reflectx.SetUnderlying(dstPtr, reflect.TypeOf((*string)(nil)))
	if dstPtr.Elem().Kind() != reflect.String {
		t.Fatal("ptr elem")
	}
	dstSlice := reflectx.NamedTypeOf("main", "S", reflect.TypeOf([]int(nil)))
	reflectx.SetUnderlying(dstSlice, reflect.TypeOf([]string(nil)))
	if dstSlice.Elem().Kind() != reflect.String {
		t.Fatal("slice elem")
	}
	dstArr := reflectx.NamedTypeOf("main", "A", reflect.TypeOf([2]int{}))
	reflectx.SetUnderlying(dstArr, reflect.TypeOf([3]byte{}))
	if dstArr.Len() != 3 || dstArr.Elem().Kind() != reflect.Uint8 {
		t.Fatal("array")
	}
	dstChan := reflectx.NamedTypeOf("main", "C", reflect.TypeOf((chan int)(nil)))
	reflectx.SetUnderlying(dstChan, reflect.TypeOf((chan string)(nil)))
	if dstChan.Elem().Kind() != reflect.String {
		t.Fatal("chan")
	}
	dstMap := reflectx.NamedTypeOf("main", "M", reflect.TypeOf(map[int]int{}))
	reflectx.SetUnderlying(dstMap, reflect.TypeOf(map[string]bool{}))
	if dstMap.Key().Kind() != reflect.String || dstMap.Elem().Kind() != reflect.Bool {
		t.Fatal("map")
	}
	dstFn0 := reflectx.NamedTypeOf("main", "F0", reflect.TypeOf(func() {}))
	reflectx.SetUnderlying(dstFn0, reflect.TypeOf(func() {}))
	for i, from := range []reflect.Type{
		reflect.TypeOf(""),
		reflect.TypeOf(float32(0)),
		reflect.TypeOf(complex64(0)),
		reflect.TypeOf([0]string{}),
		reflect.TypeOf([1]func(){}),
		reflect.TypeOf((*error)(nil)).Elem(),
	} {
		dst := reflectx.NamedTypeOf("main", "K"+string(rune('A'+i)), from)
		reflectx.SetUnderlying(dst, from)
	}
}

func TestSetElemKinds(t *testing.T) {
	ptr := reflectx.NamedTypeOf("main", "EP", reflect.TypeOf((*int)(nil)))
	reflectx.SetElem(ptr, reflect.TypeOf(""))
	if ptr.Elem().Kind() != reflect.String {
		t.Fatal("ptr")
	}
	arr := reflectx.NamedTypeOf("main", "EA", reflect.TypeOf([1]int{}))
	reflectx.SetElem(arr, reflect.TypeOf(true))
	if arr.Elem().Kind() != reflect.Bool {
		t.Fatal("array")
	}
	mp := reflectx.NamedTypeOf("main", "EM", reflect.TypeOf(map[int]int{}))
	reflectx.SetElem(mp, reflect.TypeOf(""))
	if mp.Elem().Kind() != reflect.String {
		t.Fatal("map")
	}
	ch := reflectx.NamedTypeOf("main", "EC", reflect.TypeOf((chan int)(nil)))
	reflectx.SetElem(ch, reflect.TypeOf(true))
	if ch.Elem().Kind() != reflect.Bool {
		t.Fatal("chan")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	reflectx.SetElem(reflect.TypeOf(0), reflect.TypeOf(0))
}

func TestSetValueKinds(t *testing.T) {
	set := func(dst, src interface{}) {
		dv := reflect.New(reflect.TypeOf(dst)).Elem()
		reflectx.SetValue(dv, reflect.ValueOf(src))
		if !reflect.DeepEqual(dv.Interface(), src) {
			t.Fatalf("SetValue %T: got %v want %v", src, dv.Interface(), src)
		}
	}
	set(false, true)
	set(int8(0), int8(3))
	set(uint(0), uint(7))
	set(uintptr(0), uintptr(9))
	set(float32(0), float32(1.5))
	set(complex64(0), complex64(1+2i))
	set("", "hi")
	p := unsafe.Pointer(&struct{}{})
	dv := reflect.New(reflect.TypeOf(unsafe.Pointer(nil))).Elem()
	reflectx.SetValue(dv, reflect.ValueOf(p))
	if dv.Pointer() != uintptr(p) {
		t.Fatal("unsafe.Pointer")
	}
	s := []int{1}
	dv = reflect.New(reflect.TypeOf(s)).Elem()
	reflectx.SetValue(dv, reflect.ValueOf([]int{2, 3}))
	if dv.Len() != 2 {
		t.Fatal("slice set")
	}
}

func TestSetTypeNameAndMethodX(t *testing.T) {
	typ := reflectx.NamedTypeOf("main", "Old", reflect.TypeOf(0))
	reflectx.SetTypeName(typ, "example.com/pkg", "NewName")
	if typ.Name() != "NewName" {
		t.Fatalf("name %s", typ.Name())
	}
	m := reflectx.MethodX(reflect.TypeOf((*point)(nil)), 0)
	if m.Name == "" {
		t.Fatal("MethodX")
	}
	mustPanic(t, func() { reflectx.MethodX(reflect.TypeOf((*point)(nil)), 99) })
	mustPanic(t, func() { reflectx.MethodByIndex(reflect.TypeOf((*point)(nil)), -1) })
	if _, ok := reflectx.MethodByName(reflect.TypeOf((*fmt.Stringer)(nil)).Elem(), "String"); !ok {
		t.Fatal("interface MethodByName")
	}
	if _, ok := reflectx.MethodByName(reflect.TypeOf((*point)(nil)), "missing"); ok {
		t.Fatal("missing method")
	}
	_ = reflectx.MethodByIndex(reflect.TypeOf((*fmt.Stringer)(nil)).Elem(), 0)
}

func TestFieldXErrorsAndEmbed(t *testing.T) {
	type Inner struct{ X int }
	type Outer struct{ *Inner }
	v := reflect.ValueOf(&Outer{&Inner{7}}).Elem()
	got := reflectx.FieldByIndexX(v, []int{0, 0})
	if got.Int() != 7 {
		t.Fatal(got)
	}
	if reflectx.FieldByNameX(v, "missing").IsValid() {
		t.Fatal("missing field")
	}
	if reflectx.FieldByNameFuncX(v, func(string) bool { return false }).IsValid() {
		t.Fatal("missing match")
	}
	mustPanic(t, func() { reflectx.FieldX(reflect.ValueOf(1), 0) })
	mustPanic(t, func() { reflectx.FieldByNameX(reflect.ValueOf(1), "X") })
	mustPanic(t, func() { reflectx.FieldByIndexX(reflect.ValueOf(&Outer{}).Elem(), []int{0, 0}) })
	mustPanic(t, func() { reflectx.FieldX(v, 99) })
}

func TestUpdateFieldReplaceType(t *testing.T) {
	src := reflect.TypeOf(nPoint{})
	nt := reflectx.NamedTypeOf("main", "NP", src)
	st := reflectx.StructOf([]reflect.StructField{
		{Name: "P", Type: reflect.PtrTo(src)},
		{Name: "S", Type: reflect.SliceOf(src)},
		{Name: "A", Type: reflect.ArrayOf(2, src)},
		{Name: "M", Type: reflect.MapOf(src, src)},
		{Name: "E", Type: src},
	})
	if !reflectx.UpdateField(st, map[reflect.Type]reflect.Type{src: nt}) {
		t.Fatal("UpdateField")
	}
	if reflectx.UpdateField(st, nil) {
		t.Fatal("nil rmap")
	}
	if reflectx.UpdateField(reflect.TypeOf(0), map[reflect.Type]reflect.Type{src: nt}) {
		t.Fatal("non-struct")
	}
	st2 := reflectx.StructOf([]reflect.StructField{
		{Name: "M", Type: reflect.MapOf(src, tyInt)},
	})
	if !reflectx.UpdateField(st2, map[reflect.Type]reflect.Type{src: nt}) {
		t.Fatal("map key replace")
	}
}

func TestContextResetAndAllocError(t *testing.T) {
	ctx := reflectx.NewContext()
	if ctx.IcallAlloc() != 0 {
		t.Fatal("empty alloc")
	}
	styp := reflectx.NamedStructOf("main", "CtxT", []reflect.StructField{
		{Name: "X", Type: tyInt},
	})
	typ := ctx.NewMethodSet(styp, 1, 1)
	m := reflectx.MakeMethod("M", "main", false, reflect.TypeOf(func() {}), func([]reflect.Value) []reflect.Value { return nil })
	if err := ctx.SetMethodSet(typ, []reflectx.Method{m}, false); err != nil {
		t.Fatal(err)
	}
	if ctx.IcallAlloc() == 0 {
		t.Fatal("expected icall alloc")
	}
	ctx.Reset()
	if ctx.IcallAlloc() != 0 {
		t.Fatal("reset alloc")
	}
	err := &reflectx.AllocError{Typ: tyInt, Cap: 1, Req: 2}
	if err.Error() == "" {
		t.Fatal("AllocError")
	}
	reflectx.DisableAllocateWarning = true
	defer func() { reflectx.DisableAllocateWarning = false }()
	_ = reflectx.DisableAllocateWarning
}

func TestStructOfAnonymousAndCache(t *testing.T) {
	fs := []reflect.StructField{
		{Name: "", Anonymous: true, Type: reflect.TypeOf(Point{})},
		{Name: "_", PkgPath: "main", Type: tyInt, Tag: "a"},
		{Name: "_", PkgPath: "main", Type: tyInt, Tag: "b"},
	}
	t1 := reflectx.StructOf(fs)
	t2 := reflectx.StructOf(fs)
	if t1.NumField() != 3 || !t1.Field(0).Anonymous {
		t.Fatal("anonymous")
	}
	if t2.NumField() != 3 {
		t.Fatal("struct of")
	}
	ptrAnon := reflectx.StructOf([]reflect.StructField{
		{Name: "", Anonymous: true, Type: reflect.PtrTo(reflect.TypeOf(Point{}))},
	})
	if ptrAnon.NumField() != 1 {
		t.Fatal("ptr anonymous")
	}
	if reflectx.NumMethodX(tyInt) != 0 {
		t.Fatal("int methods")
	}
	long := string(make([]byte, 200))
	_ = reflectx.StructOf([]reflect.StructField{
		{Name: "T", Type: tyInt, Tag: reflect.StructTag(long)},
	})
}

func TestRegularMemoryTypes(t *testing.T) {
	empty := reflectx.NamedStructOf("main", "Empty", nil)
	_ = reflect.New(empty).Elem().Interface()
	one := reflectx.NamedStructOf("main", "One", []reflect.StructField{
		{Name: "X", Type: tyInt},
	})
	_ = reflect.New(one).Elem().Interface()
	pad := reflectx.NamedStructOf("main", "Pad", []reflect.StructField{
		{Name: "A", Type: reflect.TypeOf(int8(0))},
		{Name: "B", Type: tyInt},
	})
	_ = reflect.New(pad).Elem().Interface()
	arr0 := reflectx.NamedTypeOf("main", "Arr0", reflect.TypeOf([0]int{}))
	_ = reflect.New(arr0).Elem().Interface()
	arrFn := reflectx.NamedTypeOf("main", "ArrFn", reflect.TypeOf([1]func(){}))
	_ = reflect.New(arrFn).Elem().Interface()
	blank1 := reflectx.NamedStructOf("main", "Blank1", []reflect.StructField{
		{Name: "_", PkgPath: "main", Type: tyInt},
	})
	_ = reflect.New(blank1).Elem().Interface()
	arrStr := reflectx.NamedTypeOf("main", "ArrStr", reflect.TypeOf([2]string{}))
	_ = reflect.New(arrStr).Elem().Interface()
	arr0fn := reflectx.NamedTypeOf("main", "Arr0Fn", reflect.TypeOf([0]func(){}))
	_ = reflect.New(arr0fn).Elem().Interface()
}

func TestStructOfCacheHit(t *testing.T) {
	fs := []reflect.StructField{
		{Name: "X", Type: tyInt},
		{Name: "Y", Type: tyInt},
	}
	t1 := reflectx.StructOf(fs)
	t2 := reflectx.StructOf(fs)
	if t1 != t2 {
		t.Fatal("expected cached struct type")
	}
}

func mustPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	fn()
}
