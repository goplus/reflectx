//go:build !js || (js && wasm)
// +build !js js,wasm

package reflectx

import (
	"fmt"
	"io"
	"reflect"
	"unsafe"
)

func toStructType(t *rtype) *structType {
	return (*structType)(unsafe.Pointer(t))
}

func toKindType(t *rtype) unsafe.Pointer {
	return unsafe.Pointer(t)
}

//go:linkname toUncommonType reflect.(*rtype).uncommon
func toUncommonType(t *rtype) *uncommonType

/*
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
*/

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

type funcTypeFixed1 struct {
	funcType
	args [1]*rtype
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

// emptyInterface is the header for an interface{} value.
type emptyInterface struct {
	typ  *rtype
	word unsafe.Pointer
}

func totype(typ reflect.Type) *rtype {
	e := (*emptyInterface)(unsafe.Pointer(&typ))
	return (*rtype)(e.word)
}

//go:nocheckptr
func (t *uncommonType) methods() []method {
	if t == nil || t.mcount == 0 {
		return nil
	}
	return (*[1 << 16]method)(add(unsafe.Pointer(t), uintptr(t.moff), "t.mcount > 0"))[:t.mcount:t.mcount]
}

//go:nocheckptr
func (t *uncommonType) exportedMethods() []method {
	if t == nil || t.xcount == 0 {
		return nil
	}
	return (*[1 << 16]method)(add(unsafe.Pointer(t), uintptr(t.moff), "t.xcount > 0"))[:t.xcount:t.xcount]
}

func tovalue(v *reflect.Value) *Value {
	return (*Value)(unsafe.Pointer(v))
}

func toValue(v Value) reflect.Value {
	return *(*reflect.Value)(unsafe.Pointer(&v))
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

// func (t *_rtype) nameOff(off nameOff) name {
// 	return name{(*byte)(resolveNameOff(unsafe.Pointer(t), int32(off)))}
// }

// type makeFuncImpl struct {
// 	code   uintptr
// 	stack  *bitVector // ptrmap for both args and results
// 	argLen uintptr    // just args
// 	ftyp   *funcType
// 	fn     func([]reflect.Value) []reflect.Value
// }

type bitVector struct {
	n    uint32 // number of bits
	data []byte
}

// funcType represents a function type.
//
// A *rtype for each in and out parameter is stored in an array that
// directly follows the funcType (and possibly its uncommonType). So
// a function type with one method, one input, and one output is:
//
//	struct {
//		funcType
//		uncommonType
//		[2]*rtype    // [0] is in, [1] is out
//	}
type funcType struct {
	rtype
	inCount  uint16
	outCount uint16 // top bit is set if last input parameter is ...
}

type uncommonFuncType struct {
	funcType
	uncommonType
	args [1]*rtype
}

func uncommonFuncTypeArgs(rt *rtype, nargs int) []*rtype {
	f := (*uncommonFuncType)(unsafe.Pointer(rt))
	return (*[1 << 16]*rtype)(unsafe.Pointer(&f.args))[:nargs:nargs]
}

func SetUnderlying(typ reflect.Type, styp reflect.Type) {
	rt := totype(typ)
	ort := totype(styp)
	switch styp.Kind() {
	case reflect.Struct:
		st := (*structType)(unsafe.Pointer(rt))
		ost := (*structType)(unsafe.Pointer(ort))
		st.fields = ost.fields
	case reflect.Ptr:
		st := (*ptrType)(unsafe.Pointer(rt))
		ost := (*ptrType)(unsafe.Pointer(ort))
		st.elem = ost.elem
	case reflect.Slice:
		st := (*sliceType)(unsafe.Pointer(rt))
		ost := (*sliceType)(unsafe.Pointer(ort))
		st.elem = ost.elem
	case reflect.Array:
		st := (*arrayType)(unsafe.Pointer(rt))
		ost := (*arrayType)(unsafe.Pointer(ort))
		st.elem = ost.elem
		st.slice = ost.slice
		st.len = ost.len
	case reflect.Chan:
		st := (*chanType)(unsafe.Pointer(rt))
		ost := (*chanType)(unsafe.Pointer(ort))
		st.elem = ost.elem
		st.dir = ost.dir
	case reflect.Interface:
		st := (*interfaceType)(unsafe.Pointer(rt))
		ost := (*interfaceType)(unsafe.Pointer(ort))
		st.methods = ost.methods
	case reflect.Map:
		st := (*mapType)(unsafe.Pointer(rt))
		ost := (*mapType)(unsafe.Pointer(ort))
		st.key = ost.key
		st.elem = ost.elem
		st.bucket = ost.bucket
		st.hasher = ost.hasher
		st.keysize = ost.keysize
		st.valuesize = ost.valuesize
		st.bucketsize = ost.bucketsize
		st.flags = ost.flags
	case reflect.Func:
		st := (*funcType)(unsafe.Pointer(rt))
		ost := (*funcType)(unsafe.Pointer(ort))
		st.inCount = ost.inCount
		st.outCount = ost.outCount
		numIn := typ.NumIn()
		numOut := typ.NumOut()
		narg := numIn + numOut
		if narg > 0 {
			args := uncommonFuncTypeArgs(rt, narg)
			var i int
			for i = 0; i < numIn; i++ {
				args[i] = totype(styp.In(i))
			}
			for j := 0; j < numOut; j++ {
				args[i+j] = totype(styp.Out(j))
			}
		}
	}
	rt.size = ort.size
	rt.tflag |= tflagUncommon | tflagExtraStar | tflagNamed
	rt.kind = ort.kind
	rt.align = ort.align
	rt.fieldAlign = ort.fieldAlign
	rt.gcdata = ort.gcdata
	rt.ptrdata = ort.ptrdata
	rt.equal = ort.equal
	//rt.str = resolveReflectName(ort.nameOff(ort.str))
	if isRegularMemory(typ) {
		rt.tflag |= tflagRegularMemory
	}
}

func newType(pkg string, name string, styp reflect.Type, mcount int, xcount int) (*rtype, []method) {
	var rt *rtype
	var fnoff uint32
	var tt reflect.Value
	ort := totype(styp)
	skind := styp.Kind()
	switch skind {
	case reflect.Struct:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(structType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*structType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*structType)(unsafe.Pointer(ort))
		st.fields = ost.fields
	case reflect.Ptr:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(ptrType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*ptrType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		st.elem = totype(styp.Elem())
	case reflect.Interface:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(interfaceType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
		}))
		st := (*interfaceType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*interfaceType)(unsafe.Pointer(ort))
		for _, m := range ost.methods {
			st.methods = append(st.methods, imethod{
				name: resolveReflectName(ost.nameOff(m.name)),
				typ:  resolveReflectType(ost.typeOff(m.typ)),
			})
		}
	case reflect.Slice:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(sliceType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*sliceType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		st.elem = totype(styp.Elem())
	case reflect.Array:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(arrayType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*arrayType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*arrayType)(unsafe.Pointer(ort))
		st.elem = ost.elem
		st.slice = ost.slice
		st.len = ost.len
	case reflect.Chan:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(chanType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*chanType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*chanType)(unsafe.Pointer(ort))
		st.elem = ost.elem
		st.dir = ost.dir
	case reflect.Func:
		numIn := styp.NumIn()
		numOut := styp.NumOut()
		narg := numIn + numOut
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(funcType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "N", Type: reflect.ArrayOf(narg, reflect.TypeOf((*rtype)(nil)))},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*funcType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*funcType)(unsafe.Pointer(ort))
		st.inCount = ost.inCount
		st.outCount = ost.outCount
		if narg > 0 {
			args := make([]*rtype, narg, narg)
			fnoff = uint32(unsafe.Sizeof((*rtype)(nil))) * uint32(narg)
			var i int
			for i = 0; i < numIn; i++ {
				args[i] = totype(styp.In(i))
			}
			for j := 0; j < numOut; j++ {
				args[i+j] = totype(styp.Out(j))
			}
			copy(tt.Elem().Field(2).Slice(0, narg).Interface().([]*rtype), args)
		}
	case reflect.Map:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(mapType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*mapType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*mapType)(unsafe.Pointer(ort))
		st.key = ost.key
		st.elem = ost.elem
		st.bucket = ost.bucket
		st.hasher = ost.hasher
		st.keysize = ost.keysize
		st.valuesize = ost.valuesize
		st.bucketsize = ost.bucketsize
		st.flags = ost.flags
	default:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(rtype{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
	}
	rt = (*rtype)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
	rt.size = ort.size
	rt.tflag = ort.tflag | tflagUncommon
	rt.kind = ort.kind
	rt.align = ort.align
	rt.fieldAlign = ort.fieldAlign
	rt.gcdata = ort.gcdata
	rt.ptrdata = ort.ptrdata
	rt.equal = ort.equal
	rt.str = resolveReflectName(ort.nameOff(ort.str))
	ut := (*uncommonType)(unsafe.Pointer(tt.Elem().Field(1).UnsafeAddr()))
	ut.mcount = uint16(mcount)
	ut.xcount = uint16(xcount)
	ut.moff = uint32(unsafe.Sizeof(uncommonType{}))
	if skind == reflect.Interface {
		return rt, nil
	} else if skind == reflect.Func {
		ut.moff += fnoff
		return rt, tt.Elem().Field(3).Slice(0, mcount).Interface().([]method)
	}
	return rt, tt.Elem().Field(2).Slice(0, mcount).Interface().([]method)
}

func NamedTypeOf(pkgpath string, name string, from reflect.Type) reflect.Type {
	rt, _ := newType(pkgpath, name, from, 0, 0)
	setTypeName(rt, pkgpath, name)
	typ := toType(rt)
	ntypeMap[typ] = &Named{Name: name, PkgPath: pkgpath, Type: typ, From: from, Kind: TkType}
	return typ
}

//go:linkname typesByString reflect.typesByString
func typesByString(s string) []*rtype

//go:linkname typelinks reflect.typelinks
func typelinks() (sections []unsafe.Pointer, offset [][]int32)

//go:linkname rtypeOff reflect.rtypeOff
func rtypeOff(section unsafe.Pointer, off int32) *rtype

func TypeLinks() []reflect.Type {
	var r []reflect.Type
	sections, offset := typelinks()
	for i, offs := range offset {
		rodata := sections[i]
		for _, off := range offs {
			typ := (*rtype)(resolveTypeOff(unsafe.Pointer(rodata), off))
			r = append(r, toType(typ))
		}
	}
	return r
}

func TypesByString(s string) []reflect.Type {
	sections, offset := typelinks()
	var ret []reflect.Type

	for offsI, offs := range offset {
		section := sections[offsI]

		// We are looking for the first index i where the string becomes >= s.
		// This is a copy of sort.Search, with f(h) replaced by (*typ[h].String() >= s).
		i, j := 0, len(offs)
		for i < j {
			h := i + (j-i)/2 // avoid overflow when computing h
			// i ≤ h < j
			typ := toType(rtypeOff(section, offs[h]))
			if !(typ.String() >= s) {
				i = h + 1 // preserves f(i-1) == false
			} else {
				j = h // preserves f(j) == true
			}
		}
		// i == j, f(i-1) == false, and f(j) (= f(i)) == true  =>  answer is i.

		// Having found the first, linear scan forward to find the last.
		// We could do a second binary search, but the caller is going
		// to do a linear scan anyway.
		for j := i; j < len(offs); j++ {
			typ := toType(rtypeOff(section, offs[j]))
			if typ.String() != s {
				break
			}
			ret = append(ret, typ)
		}
	}
	return ret
}

func DumpType(w io.Writer, typ reflect.Type) {
	rt := totype(typ)
	fmt.Fprintf(w, "%#v\n", rt)
	for _, m := range rt.methods() {
		fmt.Fprintf(w, "%v %v (%v)\t\t%#v\n",
			rt.nameOff(m.name).name(),
			rt.nameOff(m.name).pkgPath(),
			toType(rt.typeOff(m.mtyp)),
			m)
	}
}

func NumMethodX(typ reflect.Type) int {
	return totype(typ).NumMethodX()
}

func MethodX(typ reflect.Type, i int) reflect.Method {
	return totype(typ).MethodX(i)
}

func (t *rtype) NumMethodX() int {
	return len(t.methods())
}

func (t *rtype) MethodX(i int) (m reflect.Method) {
	if t.Kind() == reflect.Interface {
		return toType(t).Method(i)
	}
	methods := t.methods()
	if i < 0 || i >= len(methods) {
		panic("reflect: Method index out of range")
	}
	p := methods[i]
	pname := t.nameOff(p.name)
	m.Name = pname.name()
	m.Index = i
	fl := flag(reflect.Func)
	if t.tflag&tflagUserMethod != 0 {
		fl |= flagIndir
	}
	mtyp := t.typeOff(p.mtyp)
	if mtyp == nil {
		return
	}
	ft := (*funcType)(unsafe.Pointer(mtyp))
	in := make([]reflect.Type, 0, 1+len(ft.in()))
	in = append(in, toType(t))
	for _, arg := range ft.in() {
		in = append(in, toType(arg))
	}
	out := make([]reflect.Type, 0, len(ft.out()))
	for _, ret := range ft.out() {
		out = append(out, toType(ret))
	}
	mt := reflect.FuncOf(in, out, ft.IsVariadic())
	m.Type = mt
	tfn := t.textOff(p.tfn)
	fn := unsafe.Pointer(&tfn)
	m.Func = toValue(Value{totype(mt), fn, fl})
	return m
}

func (t *rtype) MethodByNameX(name string) (m reflect.Method, ok bool) {
	if t.Kind() == reflect.Interface {
		return toType(t).MethodByName(name)
	}
	if ut := t.uncommon(); ut != nil {
		for i, p := range ut.methods() {
			if t.nameOff(p.name).name() == name {
				return t.MethodX(i), true
			}
		}
	}
	return reflect.Method{}, false
}

// Field returns the i'th field of the struct v.
// It panics if v's Kind is not Struct or i is out of range.
func FieldX(v reflect.Value, i int) reflect.Value {
	mustBe("reflect.Value.Field", v, reflect.Struct)
	rv := tovalue(&v)
	tt := (*structType)(unsafe.Pointer(rv.typ))
	if uint(i) >= uint(len(tt.fields)) {
		panic("reflect: Field index out of range")
	}
	field := &tt.fields[i]
	typ := field.typ

	// Inherit permission bits from v, but clear flagEmbedRO.
	fl := rv.flag&(flagStickyRO|flagIndir|flagAddr) | flag(typ.Kind())
	// Using an unexported field forces flagRO.
	// if !field.name.isExported() {
	// 	if field.embedded() {
	// 		fl |= flagEmbedRO
	// 	} else {
	// 		fl |= flagStickyRO
	// 	}
	// }
	// Either flagIndir is set and v.ptr points at struct,
	// or flagIndir is not set and v.ptr is the actual struct data.
	// In the former case, we want v.ptr + offset.
	// In the latter case, we must have field.offset = 0,
	// so v.ptr + field.offset is still the correct address.
	ptr := add(rv.ptr, field.offset(), "same as non-reflect &v.field")
	return toValue(Value{typ, ptr, fl})
}
