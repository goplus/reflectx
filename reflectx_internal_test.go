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

func TestLLGoSetUnderlyingPreservesHash(t *testing.T) {
	if runtime.Compiler != "llgo" {
		t.Skip("llgo ABI hashes")
	}
	realType := reflect.TypeOf(struct{ Value int }{})
	hashes := make(map[uint32]bool)
	for _, name := range []string{"PlaceholderA", "PlaceholderB"} {
		typ := NamedTypeOf("hashprobe", name, reflect.TypeOf(struct{}{}))
		hash := totype(typ).Hash
		ptr := reflect.PointerTo(typ)
		pointerHash := totype(ptr).Hash
		function := reflect.FuncOf([]reflect.Type{ptr}, nil, false)
		SetUnderlying(typ, realType)
		if got := totype(typ).Hash; got != hash {
			t.Fatalf("%s: underlying update changed hash from %x to %x", name, hash, got)
		}
		if got := totype(ptr).Hash; got != pointerHash || got != hash*16777619^uint32('*') {
			t.Fatalf("%s: pointer hash changed or is inconsistent", name)
		}
		if reflect.PointerTo(typ) != ptr || reflect.FuncOf([]reflect.Type{ptr}, nil, false) != function {
			t.Fatal("underlying update changed cached type identity")
		}
		v := reflect.New(typ).Elem()
		v.Field(0).SetInt(42)
		key := v.Interface()
		m := map[interface{}]int{key: 7}
		if m[key] != 7 {
			t.Fatal("map lookup after underlying update")
		}
		SetTypeName(typ, "hashprobe/renamed", name)
		if totype(typ).Hash != hash || totype(ptr).Hash != pointerHash {
			t.Fatal("renaming changed published hashes")
		}
		hashes[hash] = true
	}
	if len(hashes) != 2 {
		t.Fatal("different names over the same underlying type have equal hashes")
	}
}

func TestGoDynamicTypeHashes(t *testing.T) {
	if runtime.Compiler == "llgo" {
		t.Skip("Go ABI hashes")
	}
	static := reflect.TypeOf(dynamicHashStatic(0))
	clone := NewMethodSet(static, 0, 1)
	if totype(clone).Hash != totype(static).Hash || totype(reflect.PointerTo(clone)).Hash != totype(reflect.PointerTo(static)).Hash {
		t.Fatal("static clone hash changed")
	}
	hashes := map[uint32]bool{}
	for _, pkg := range []string{"main", "example/a/shared", "example/b/shared"} {
		for _, name := range []string{"T", "Box[int]", "T$1"} {
			typ := NamedTypeOf(pkg, name, reflect.TypeOf(0))
			if got, want := totype(typ).Hash, dynamicNamedTypeHash(pkg, name); got != want {
				t.Fatalf("%s.%s hash=%x, want %x", pkg, name, got, want)
			}
			hashes[totype(typ).Hash] = true
			ptr := reflect.PointerTo(typ)
			if totype(ptr).Hash != totype(typ).Hash*16777619^uint32('*') {
				t.Fatal("pointer hash")
			}
			clone := NewMethodSet(typ, 0, 1)
			if totype(clone).Hash != totype(typ).Hash || totype(reflect.PointerTo(clone)).Hash != totype(ptr).Hash {
				t.Fatal("dynamic clone hash changed")
			}
			f := reflect.FuncOf([]reflect.Type{ptr}, nil, false)
			if reflect.FuncOf([]reflect.Type{ptr}, nil, false) != f {
				t.Fatal("unstable FuncOf")
			}
			v := reflect.New(clone).Elem()
			v.SetInt(42)
			key := v.Interface()
			if map[interface{}]int{key: 7}[key] != 7 {
				t.Fatal("dynamic map key")
			}
		}
	}
	if len(hashes) != 9 {
		t.Fatal("unexpected hash collisions")
	}
	iface := NewInterfaceType("hashprobe", "Interface")
	if totype(iface).Hash != dynamicNamedTypeHash("hashprobe", "Interface") {
		t.Fatal("interface hash")
	}
	if err := SetInterfaceType(iface, nil, nil); err != nil {
		t.Fatal(err)
	}
	if totype(iface).Hash != dynamicNamedTypeHash("hashprobe", "Interface") {
		t.Fatal("interface update changed hash")
	}
	typ := NamedTypeOf("hashprobe", "Placeholder", reflect.TypeOf(struct{}{}))
	hash := totype(typ).Hash
	ptr := reflect.PointerTo(typ)
	phash := totype(ptr).Hash
	SetUnderlying(typ, reflect.TypeOf(struct{ Value int }{}))
	SetTypeName(typ, "hashprobe/renamed", "Renamed")
	if totype(typ).Hash != hash || totype(ptr).Hash != phash {
		t.Fatal("underlying update or rename changed hash")
	}
}
