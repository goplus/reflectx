// +build !js js,wasm

package reflectx

import (
	"reflect"
	"unsafe"
)

func toStructType(t *rtype) *structType {
	return (*structType)(unsafe.Pointer(t))
}

func toUncommonType(t *rtype) *uncommonType {
	if t.tflag&tflagUncommon == 0 {
		return nil
	}
	switch t.Kind() {
	case reflect.Struct:
		return &(*structTypeUncommon)(unsafe.Pointer(t)).u
	case reflect.Ptr:
		type u struct {
			ptrType
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case reflect.Func:
		type u struct {
			funcType
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case reflect.Slice:
		type u struct {
			sliceType
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case reflect.Array:
		type u struct {
			arrayType
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case reflect.Chan:
		type u struct {
			chanType
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case reflect.Map:
		type u struct {
			mapType
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	case reflect.Interface:
		type u struct {
			interfaceType
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	default:
		type u struct {
			rtype
			u uncommonType
		}
		return &(*u)(unsafe.Pointer(t)).u
	}
}

// uncommonType is present only for defined types or types with methods
// (if T is a defined type, the uncommonTypes for T and *T have methods).
// Using a pointer to this struct reduces the overall size required
// to describe a non-defined type with no methods.
type uncommonType struct {
	pkgPath nameOff // import path; empty for built-in types like int, string
	mcount  uint16  // number of methods
	xcount  uint16  // number of exported methods
	moff    uint32  // offset from this uncommontype to [mcount]method
	_       uint32  // unused
}

type funcTypeFixed4 struct {
	funcType
	args [4]*rtype
}
type funcTypeFixed8 struct {
	funcType
	args [8]*rtype
}
type funcTypeFixed16 struct {
	funcType
	args [16]*rtype
}
type funcTypeFixed32 struct {
	funcType
	args [32]*rtype
}
type funcTypeFixed64 struct {
	funcType
	args [64]*rtype
}
type funcTypeFixed128 struct {
	funcType
	args [128]*rtype
}

func NamedTypeOf(pkgpath string, name string, from reflect.Type) (typ reflect.Type) {
	switch from.Kind() {
	case reflect.Array:
		typ = reflect.ArrayOf(from.Len(), emptyType())
		dst := totype(typ)
		src := totype(from)
		copyType(dst, src)
		d := (*arrayType)(unsafe.Pointer(dst))
		s := (*arrayType)(unsafe.Pointer(src))
		d.elem = s.elem
		d.slice = s.slice
		d.len = s.len
		setTypeName(dst, pkgpath, name)
	case reflect.Slice:
		typ = reflect.SliceOf(emptyType())
		dst := totype(typ)
		src := totype(from)
		copyType(dst, src)
		d := (*sliceType)(unsafe.Pointer(dst))
		s := (*sliceType)(unsafe.Pointer(src))
		d.elem = s.elem
		setTypeName(dst, pkgpath, name)
	case reflect.Map:
		typ = reflect.MapOf(emptyType(), emptyType())
		dst := totype(typ)
		src := totype(from)
		copyType(dst, src)
		d := (*mapType)(unsafe.Pointer(dst))
		s := (*mapType)(unsafe.Pointer(src))
		d.key = s.key
		d.elem = s.elem
		d.bucket = s.bucket
		d.hasher = s.hasher
		d.keysize = s.keysize
		d.valuesize = s.valuesize
		d.bucketsize = s.bucketsize
		d.flags = s.flags
		dst.str = resolveReflectName(newName(name, "", isExported(name)))
		setTypeName(dst, pkgpath, name)
	case reflect.Ptr:
		typ = reflect.PtrTo(emptyType())
		dst := totype(typ)
		src := totype(from)
		copyType(dst, src)
		d := (*ptrType)(unsafe.Pointer(dst))
		s := (*ptrType)(unsafe.Pointer(src))
		d.elem = s.elem
		setTypeName(dst, pkgpath, name)
	case reflect.Chan:
		typ = reflect.ChanOf(from.ChanDir(), emptyType())
		dst := totype(typ)
		src := totype(from)
		copyType(dst, src)
		d := (*chanType)(unsafe.Pointer(dst))
		s := (*chanType)(unsafe.Pointer(src))
		d.elem = s.elem
		d.dir = s.dir
		setTypeName(dst, pkgpath, name)
	case reflect.Func:
		numIn := from.NumIn()
		in := make([]reflect.Type, numIn, numIn)
		for i := 0; i < numIn; i++ {
			in[i] = from.In(i)
		}
		numOut := from.NumOut()
		out := make([]reflect.Type, numOut, numOut)
		for i := 0; i < numOut; i++ {
			out[i] = from.Out(i)
		}
		out = append(out, emptyType())
		typ = reflect.FuncOf(in, out, from.IsVariadic())
		dst := totype(typ)
		src := totype(from)
		d := (*funcType)(unsafe.Pointer(dst))
		s := (*funcType)(unsafe.Pointer(src))
		d.inCount = s.inCount
		d.outCount = s.outCount
		setTypeName(dst, pkgpath, name)
	default:
		var fields []reflect.StructField
		if from.Kind() == reflect.Struct {
			for i := 0; i < from.NumField(); i++ {
				fields = append(fields, from.Field(i))
			}
		}
		fields = append(fields, reflect.StructField{
			Name: hashName(pkgpath, name),
			Type: typEmptyStruct,
		})
		typ = StructOf(fields)
		rt := totype(typ)
		st := toStructType(rt)
		st.fields = st.fields[:len(st.fields)-1]
		copyType(rt, totype(from))
		setTypeName(rt, pkgpath, name)
	}
	nt := &Named{Name: name, PkgPath: pkgpath, Type: typ, From: from, Kind: TkType}
	ntypeMap[typ] = nt
	return typ
}

func totype(typ reflect.Type) *rtype {
	v := reflect.Zero(typ)
	rt := (*Value)(unsafe.Pointer(&v)).typ
	return rt
}

func (t *uncommonType) methods() []method {
	if t.mcount == 0 {
		return nil
	}
	return (*[1 << 16]method)(add(unsafe.Pointer(t), uintptr(t.moff), "t.mcount > 0"))[:t.mcount:t.mcount]
}

func (t *uncommonType) exportedMethods() []method {
	if t.xcount == 0 {
		return nil
	}
	return (*[1 << 16]method)(add(unsafe.Pointer(t), uintptr(t.moff), "t.xcount > 0"))[:t.xcount:t.xcount]
}

func tovalue(v *reflect.Value) *Value {
	return (*Value)(unsafe.Pointer(v))
}

func (t *rtype) uncommon() *uncommonType {
	return toUncommonType(t)
}

func (t *rtype) exportedMethods() []method {
	ut := t.uncommon()
	if ut == nil {
		return nil
	}
	return ut.exportedMethods()
}

func (t *rtype) methods() []method {
	ut := t.uncommon()
	if ut == nil {
		return nil
	}
	return ut.methods()
}

func (t *funcType) in() []*rtype {
	uadd := unsafe.Sizeof(*t)
	if t.tflag&tflagUncommon != 0 {
		uadd += unsafe.Sizeof(uncommonType{})
	}
	if t.inCount == 0 {
		return nil
	}
	return (*[1 << 20]*rtype)(add(unsafe.Pointer(t), uadd, "t.inCount > 0"))[:t.inCount:t.inCount]
}

func (t *funcType) out() []*rtype {
	uadd := unsafe.Sizeof(*t)
	if t.tflag&tflagUncommon != 0 {
		uadd += unsafe.Sizeof(uncommonType{})
	}
	outCount := t.outCount & (1<<15 - 1)
	if outCount == 0 {
		return nil
	}
	return (*[1 << 20]*rtype)(add(unsafe.Pointer(t), uadd, "outCount > 0"))[t.inCount : t.inCount+outCount : t.inCount+outCount]
}

func (t *rtype) IsVariadic() bool {
	if t.Kind() != reflect.Func {
		panic("reflect: IsVariadic of non-func type " + toType(t).String())
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return tt.outCount&(1<<15) != 0
}

type makeFuncImpl struct {
	code   uintptr
	stack  *bitVector // ptrmap for both args and results
	argLen uintptr    // just args
	ftyp   *funcType
	fn     func([]reflect.Value) []reflect.Value
}

type bitVector struct {
	n    uint32 // number of bits
	data []byte
}
