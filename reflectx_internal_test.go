package reflectx

import (
	"fmt"
	"reflect"
	"runtime"
	"testing"
)

func TestUnexportedHelpers(t *testing.T) {
	t1 := reflect.TypeOf(struct {
		A int
		B string
	}{})
	t2 := reflect.TypeOf(struct {
		A int
		B string
	}{})
	if !checkFields(t1, t2) {
		t.Fatal("identical")
	}
	if checkFields(t1, reflect.TypeOf(struct{ A int }{})) {
		t.Fatal("num field")
	}
	if checkFields(t1, reflect.TypeOf(struct {
		A int
		C string
	}{})) {
		t.Fatal("name")
	}
	if checkFields(t1, reflect.TypeOf(struct {
		A int8
		B string
	}{})) {
		t.Fatal("type")
	}
	anon := reflect.TypeOf(struct{ int }{})
	named := reflect.TypeOf(struct{ X int }{})
	if checkFields(anon, named) {
		t.Fatal("anonymous")
	}

	pt := reflect.PtrTo(t1)
	if toElem(pt) != t1 {
		t.Fatal("toElem ptr")
	}
	if toElem(t1) != t1 {
		t.Fatal("toElem")
	}
	v := reflect.New(t1)
	if toElemValue(v).Type() != t1 {
		t.Fatal("toElemValue ptr")
	}
	if toElemValue(v.Elem()).Type() != t1 {
		t.Fatal("toElemValue")
	}
	_ = extraFieldMethod(0, reflect.TypeOf(helper(0)), map[string]bool{"String": true})
	_ = extraFieldMethod(0, reflect.PtrTo(reflect.TypeOf(helper(0))), nil)
}

type helper int

func (helper) String() string { return "" }

func (h *helper) Set(v int) { *h = helper(v) }

type dynamicHashStatic int

func TestLLGoDynamicTypeHashes(t *testing.T) {
	if runtime.Compiler != "llgo" {
		t.Skip("llgo ABI hashes")
	}
	static := reflect.TypeOf(dynamicHashStatic(0))
	staticClone := NewMethodSet(static, 0, 0)
	if totype(staticClone).Hash != totype(static).Hash || totype(reflect.PointerTo(staticClone)).Hash != totype(reflect.PointerTo(static)).Hash {
		t.Fatal("static method-set clone changed hash")
	}
	hashes := map[uint32]bool{}
	for _, pkg := range []string{"main", "example.org/a/shared", "example.org/b/shared"} {
		for _, name := range []string{"T", "Box[int]", "T$1"} {
			typ := NamedTypeOf(pkg, name, reflect.TypeOf(int(0)))
			if got, want := totype(typ).Hash, func() uint32 {
				h := uint32(2166136261)
				for _, v := range []byte(pkg + "\x00" + name) {
					h = h*16777619 ^ uint32(v)
				}
				return h
			}(); got != want {
				t.Fatalf("%s.%s: hash %x, want %x", pkg, name, got, want)
			}
			hashes[totype(typ).Hash] = true
			ptr := reflect.PointerTo(typ)
			if got, want := totype(ptr).Hash, totype(typ).Hash*16777619^uint32('*'); got != want {
				t.Fatalf("pointer hash %x, want %x", got, want)
			}
			cloned := NewMethodSet(typ, 0, 0)
			if totype(cloned).Hash != totype(typ).Hash || totype(reflect.PointerTo(cloned)).Hash != totype(ptr).Hash {
				t.Fatal("method-set clone changed hash")
			}
			f := reflect.FuncOf([]reflect.Type{ptr}, []reflect.Type{typ}, false)
			if reflect.FuncOf([]reflect.Type{ptr}, []reflect.Type{typ}, false) != f {
				t.Fatal("unstable FuncOf")
			}
			v := reflect.New(typ).Elem()
			v.SetInt(42)
			m := map[interface{}]int{v.Interface(): 7}
			if m[v.Interface()] != 7 {
				t.Fatal("dynamic map key lookup")
			}
		}
	}
	if len(hashes) != 9 {
		t.Fatalf("unexpected hash collisions: %d", len(hashes))
	}
	for i, from := range []reflect.Type{reflect.TypeOf(struct{ N int }{}), reflect.TypeOf([]int{}), reflect.TypeOf(map[string]int{}), reflect.TypeOf((chan int)(nil)), reflect.TypeOf((func(int) int)(nil))} {
		typ := NamedTypeOf("hashprobe", fmt.Sprintf("Kind%d", i), from)
		clone := NewMethodSet(typ, 0, 0)
		if totype(typ).Hash == 0 || totype(clone).Hash != totype(typ).Hash {
			t.Fatalf("%v: invalid cloned hash", typ)
		}
		if typ.Kind() == reflect.Func {
			if typ.NumIn() != 1 || typ.In(0) != reflect.TypeOf(0) || typ.NumOut() != 1 || typ.Out(0) != reflect.TypeOf(0) {
				t.Fatal("named function signature changed")
			}
			fn := reflect.ValueOf(func(v int) int { return v + 1 }).Convert(typ)
			if got := fn.Call([]reflect.Value{reflect.ValueOf(41)})[0].Int(); got != 42 {
				t.Fatalf("named function call = %d", got)
			}
			clonedFn := fn.Convert(clone)
			if got := clonedFn.Call([]reflect.Value{reflect.ValueOf(42)})[0].Int(); got != 43 {
				t.Fatalf("cloned function call = %d", got)
			}
		}
	}
}
