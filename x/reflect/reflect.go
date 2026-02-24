package reflect

import (
	"reflect"
	"runtime"
	"unsafe"

	"github.com/goplus/reflectx/internal/abi"
)

// memmove copies size bytes to dst from src. No write barriers are used.
//
//go:noescape
//go:linkname memmove reflect.memmove
func memmove(dst, src unsafe.Pointer, size uintptr)

// typedmemmove copies a value of type t to dst from src.
//
//go:noescape
//go:linkname typedmemmove reflect.typedmemmove
func typedmemmove(t *abi.Type, dst, src unsafe.Pointer)

// resolveTypeOff resolves an *rtype offset from a base type.
// The (*rtype).typeOff method is a convenience wrapper for this function.
// Implemented in the runtime package.
//
//go:linkname resolveTypeOff reflect.resolveTypeOff
func resolveTypeOff(rtype unsafe.Pointer, off int32) unsafe.Pointer

// resolveTextOff resolves a function pointer offset from a base type.
// The (*rtype).textOff method is a convenience wrapper for this function.
// Implemented in the runtime package.
//
//go:linkname resolveTextOff reflect.resolveTextOff
func resolveTextOff(rtype unsafe.Pointer, off int32) unsafe.Pointer

// addReflectOff adds a pointer to the reflection lookup map in the runtime.
// It returns a new ID that can be used as a typeOff or textOff, and will
// be resolved correctly. Implemented in the runtime package.
//
//go:linkname addReflectOff reflect.addReflectOff
func addReflectOff(ptr unsafe.Pointer) int32

// resolveReflectName adds a name to the reflection lookup map in the runtime.
// It returns a new nameOff that can be used to refer to the pointer.
func resolveReflectName(n abi.Name) abi.NameOff {
	return abi.NameOff(addReflectOff(unsafe.Pointer(n.Bytes)))
}

//go:linkname resolveNameOff reflect.resolveNameOff
func resolveNameOff(ptrInModule unsafe.Pointer, off int32) unsafe.Pointer

func newName(n, tag string, exported, embedded bool) abi.Name {
	return abi.NewName(n, tag, exported, embedded)
}

func stringFor(t *abi.Type) string {
	s := nameOff(t, t.Str).Name()
	if t.TFlag&abi.TFlagExtraStar != 0 {
		return s[1:]
	}
	return s
}

func nameFor(t *abi.Type) string {
	if !t.HasName() {
		return ""
	}
	s := stringFor(t)
	i := len(s) - 1
	sqBrackets := 0
	for i >= 0 && (s[i] != '.' || sqBrackets != 0) {
		switch s[i] {
		case ']':
			sqBrackets++
		case '[':
			sqBrackets--
		}
		i--
	}
	return s[i+1:]
}

func pkgPathFor(t *abi.Type) string {
	if t.TFlag&abi.TFlagNamed == 0 {
		return ""
	}
	ut := t.Uncommon()
	if ut == nil {
		return ""
	}
	return nameOff(t, ut.PkgPath).Name()
}

//go:noescape
//go:linkname unsafe_New reflect.unsafe_New
func unsafe_New(*abi.Type) unsafe.Pointer

//go:noescape
//go:linkname unsafe_NewArray reflect.unsafe_NewArray
func unsafe_NewArray(*abi.Type, int) unsafe.Pointer

func Zero(t *abi.Type) Value {
	if t == nil {
		panic("reflect: Zero(nil)")
	}
	fl := flag(t.Kind())
	if !t.IsDirectIface() {
		var p unsafe.Pointer
		if t.Size() <= abi.ZeroValSize {
			p = unsafe.Pointer(&zeroVal[0])
		} else {
			p = unsafe_New(t)
		}
		return Value{t, p, fl | flagIndir}
	}
	return Value{t, nil, fl}
}

//go:linkname zeroVal runtime.zeroVal
var zeroVal [abi.ZeroValSize]byte

func add(p unsafe.Pointer, x uintptr, whySafe string) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

// funcName returns the name of f, for use in error messages.
func funcName(f func([]Value) []Value) string {
	pc := *(*uintptr)(unsafe.Pointer(&f))
	rf := runtime.FuncForPC(pc)
	if rf != nil {
		return rf.Name()
	}
	return "closure"
}

func toUncommonType(t *abi.Type) *abi.UncommonType {
	return t.Uncommon()
}

type assignToFunc = func(v Value, context string, dst *abi.Type, target unsafe.Pointer) Value

var assignTo assignToFunc

func nameOff(t *abi.Type, off abi.NameOff) abi.Name {
	return abi.Name{Bytes: (*byte)(resolveNameOff(unsafe.Pointer(t), int32(off)))}
}

func typeOff(t *abi.Type, off abi.TypeOff) *abi.Type {
	return (*abi.Type)(resolveTypeOff(unsafe.Pointer(t), int32(off)))
}

func init() {
	v := reflect.ValueOf(func() {})
	v.Call(nil)
	t := abi.TypeOf(v)
	for _, m := range t.Uncommon().Methods() {
		if nameOff(t, m.Name).Name() == "assignTo" {
			if m.Tfn != -1 {
				tfn := resolveTextOff(unsafe.Pointer(t), int32(m.Tfn))
				*(*unsafe.Pointer)(unsafe.Pointer(&assignTo)) = unsafe.Pointer(&tfn)
			}
			break
		}
	}
	if assignTo == nil {
		panic("reflectx: failed to resolve reflect.Value.assignTo")
	}
}

func haveIdenticalType(T, V *abi.Type, cmpTags bool) bool {
	if cmpTags {
		return T == V
	}

	if nameFor(T) != nameFor(V) || T.Kind() != V.Kind() || pkgPathFor(T) != pkgPathFor(V) {
		return false
	}

	return haveIdenticalUnderlyingType(T, V, false)
}

func haveIdenticalUnderlyingType(T, V *abi.Type, cmpTags bool) bool {
	if T == V {
		return true
	}

	kind := Kind(T.Kind())
	if kind != Kind(V.Kind()) {
		return false
	}

	// Non-composite types of equal kind have same underlying type
	// (the predefined instance of the type).
	if Bool <= kind && kind <= Complex128 || kind == String || kind == UnsafePointer {
		return true
	}

	// Composite types.
	switch kind {
	case Array:
		return T.Len() == V.Len() && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Chan:
		return V.ChanDir() == T.ChanDir() && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Func:
		t := (*funcType)(unsafe.Pointer(T))
		v := (*funcType)(unsafe.Pointer(V))
		if t.OutCount != v.OutCount || t.InCount != v.InCount {
			return false
		}
		for i := 0; i < t.NumIn(); i++ {
			if !haveIdenticalType(t.In(i), v.In(i), cmpTags) {
				return false
			}
		}
		for i := 0; i < t.NumOut(); i++ {
			if !haveIdenticalType(t.Out(i), v.Out(i), cmpTags) {
				return false
			}
		}
		return true

	case Interface:
		t := (*interfaceType)(unsafe.Pointer(T))
		v := (*interfaceType)(unsafe.Pointer(V))
		if len(t.Methods) == 0 && len(v.Methods) == 0 {
			return true
		}
		// Might have the same methods but still
		// need a run time conversion.
		return false

	case Map:
		return haveIdenticalType(T.Key(), V.Key(), cmpTags) && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Pointer, Slice:
		return haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Struct:
		t := (*structType)(unsafe.Pointer(T))
		v := (*structType)(unsafe.Pointer(V))
		if len(t.Fields) != len(v.Fields) {
			return false
		}
		if t.PkgPath.Name() != v.PkgPath.Name() {
			return false
		}
		for i := range t.Fields {
			tf := &t.Fields[i]
			vf := &v.Fields[i]
			if tf.Name.Name() != vf.Name.Name() {
				return false
			}
			if !haveIdenticalType(tf.Typ, vf.Typ, cmpTags) {
				return false
			}
			if cmpTags && tf.Name.Tag() != vf.Name.Tag() {
				return false
			}
			if tf.Offset != vf.Offset {
				return false
			}
			if tf.Embedded() != vf.Embedded() {
				return false
			}
		}
		return true
	}

	return false
}

type itab = abi.ITab

type iface struct {
	tab  *itab
	data unsafe.Pointer
}

type eface struct {
	_type *abi.Type
	data  unsafe.Pointer
}

func interequal(p, q unsafe.Pointer) bool {
	x := *(*iface)(p)
	y := *(*iface)(q)
	return x.tab == y.tab && ifaceeq(x.tab, x.data, y.data)
}
func ifaceeq(tab *itab, x, y unsafe.Pointer) bool {
	if tab == nil {
		return true
	}
	t := tab.Type
	eq := t.Equal
	if eq == nil {
		panic(errorString("comparing uncomparable type " + stringFor(t)))
	}
	if t.IsDirectIface() {
		// See comment in efaceeq.
		return x == y
	}
	return eq(x, y)
}

// An errorString represents a runtime error described by a single string.
type errorString string

func (e errorString) RuntimeError() {}

func (e errorString) Error() string {
	return "runtime error: " + string(e)
}
