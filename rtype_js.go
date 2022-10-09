//go:build js && !wasm
// +build js,!wasm

package reflectx

import (
	"reflect"
	"unsafe"

	"github.com/gopherjs/gopherjs/js"
)

//go:linkname reflectType reflect.reflectType
func reflectType(typ *js.Object) *rtype

//go:linkname setKindType reflect.setKindType
func setKindType(rt *rtype, kindType interface{})

//go:linkname newNameOff reflect.newNameOff
func newNameOff(n name) nameOff

//go:linkname newTypeOff reflect.newTypeOff
func newTypeOff(rt *rtype) typeOff

//go:linkname makeValue reflect.makeValue
func makeValue(t *rtype, v *js.Object, fl flag) reflect.Value

// func jsType(typ Type) *js.Object {
// 	return js.InternalObject(typ).Get("jsType")
// }

// func reflectType(typ *js.Object) *rtype {
// 	return _reflectType(typ, internalStr)
// }

func toStructType(t *rtype) *structType {
	kind := js.InternalObject(t).Get("kindType")
	return (*structType)(unsafe.Pointer(kind.Unsafe()))
}

func toKindType(t *rtype) unsafe.Pointer {
	return unsafe.Pointer(js.InternalObject(t).Get("kindType").Unsafe())
}

func toUncommonType(t *rtype) *uncommonType {
	kind := js.InternalObject(t).Get("uncommonType")
	if kind == js.Undefined {
		return nil
	}
	return (*uncommonType)(unsafe.Pointer(kind.Unsafe()))
}

type uncommonType struct {
	pkgPath nameOff
	mcount  uint16
	xcount  uint16
	moff    uint32

	_methods []method
}

func (t *uncommonType) exportedMethods() []method {
	if t.xcount == 0 {
		return nil
	}
	return t._methods[:t.xcount:t.xcount]
}

func (t *uncommonType) methods() []method {
	if t.mcount == 0 {
		return nil
	}
	return t._methods
}

func (t *rtype) ptrTo() *rtype {
	return reflectType(js.Global.Call("$ptrType", jsType(t)))
}

//go:linkename rtype_uncommon reflect.(*rtype).uncommon
func rtype_uncommon(*rtype) *uncommonType

//go:linkename rtype_methods reflect.(*rtype).methods
func rtype_methods(*rtype) []method

func (t *rtype) uncommon() *uncommonType {
	return rtype_uncommon(t)
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
	return ut._methods
}

func (t *rtype) IsVariadic() bool {
	if t.Kind() != reflect.Func {
		panic("reflect: IsVariadic of non-func type " + toType(t).String())
	}
	tt := (*funcType)(unsafe.Pointer(t))
	return tt.outCount&(1<<15) != 0
}

/*
var $kindBool = 1;
var $kindInt = 2;
var $kindInt8 = 3;
var $kindInt16 = 4;
var $kindInt32 = 5;
var $kindInt64 = 6;
var $kindUint = 7;
var $kindUint8 = 8;
var $kindUint16 = 9;
var $kindUint32 = 10;
var $kindUint64 = 11;
var $kindUintptr = 12;
var $kindFloat32 = 13;
var $kindFloat64 = 14;
var $kindComplex64 = 15;
var $kindComplex128 = 16;
var $kindArray = 17;
var $kindChan = 18;
var $kindFunc = 19;
var $kindInterface = 20;
var $kindMap = 21;
var $kindPtr = 22;
var $kindSlice = 23;
var $kindString = 24;
var $kindStruct = 25;
var $kindUnsafePointer = 26;

var $Bool          = $newType( 1, $kindBool,          "bool",           true, "", false, null);
var $Int           = $newType( 4, $kindInt,           "int",            true, "", false, null);
var $Int8          = $newType( 1, $kindInt8,          "int8",           true, "", false, null);
var $Int16         = $newType( 2, $kindInt16,         "int16",          true, "", false, null);
var $Int32         = $newType( 4, $kindInt32,         "int32",          true, "", false, null);
var $Int64         = $newType( 8, $kindInt64,         "int64",          true, "", false, null);
var $Uint          = $newType( 4, $kindUint,          "uint",           true, "", false, null);
var $Uint8         = $newType( 1, $kindUint8,         "uint8",          true, "", false, null);
var $Uint16        = $newType( 2, $kindUint16,        "uint16",         true, "", false, null);
var $Uint32        = $newType( 4, $kindUint32,        "uint32",         true, "", false, null);
var $Uint64        = $newType( 8, $kindUint64,        "uint64",         true, "", false, null);
var $Uintptr       = $newType( 4, $kindUintptr,       "uintptr",        true, "", false, null);
var $Float32       = $newType( 4, $kindFloat32,       "float32",        true, "", false, null);
var $Float64       = $newType( 8, $kindFloat64,       "float64",        true, "", false, null);
var $Complex64     = $newType( 8, $kindComplex64,     "complex64",      true, "", false, null);
var $Complex128    = $newType(16, $kindComplex128,    "complex128",     true, "", false, null);
var $String        = $newType( 8, $kindString,        "string",         true, "", false, null);
var $UnsafePointer = $newType( 4, $kindUnsafePointer, "unsafe.Pointer", true, "", false, null);
*/
//var $newType = function(size, kind, string, named, pkg, exported, constructor) {

var (
	fnNewType = js.Global.Get("$newType")
)

/*
	Invalid Kind = iota
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	Array
	Chan
	Func
	Interface
	Map
	Ptr
	Slice
	String
	Struct
	UnsafePointer
*/
var (
	sizes = []int{0, 1,
		4, 1, 2, 4, 8, // int
		4, 1, 2, 4, 8, // uint
		4,    // uintptr
		4, 8, // float
		8, 16, // complex
		4, // array
		4, //
		4,
		4,
		4,
		4,
		12, // slice
		8,  // string
		4,  // struct
		4,  // UnsafePointer
	}
)

func tovalue(v *reflect.Value) *Value {
	return (*Value)(unsafe.Pointer(v))
}

func toValue(v Value) reflect.Value {
	return *(*reflect.Value)(unsafe.Pointer(&v))
}

func NamedTypeOf(pkg string, name string, from reflect.Type) (typ reflect.Type) {
	rt, _ := newType(pkg, name, from, 0, 0)
	setTypeName(rt, pkg, name)
	typ = toType(rt)
	nt := &Named{Name: name, PkgPath: pkg, Type: typ, From: from, Kind: TkType}
	ntypeMap[typ] = nt
	return
}

var (
	jsUncommonTyp = js.InternalObject(reflect.TypeOf((*rtype)(nil))).Get("uncommonType").Get("constructor")
)

func resetUncommonType(rt *rtype, mcount int, xcount int) *uncommonType {
	ut := jsUncommonTyp.New()
	v := js.InternalObject(ut).Get("_methods").Get("constructor")
	ut.Set("xcount", xcount)
	ut.Set("mcount", mcount)
	ut.Set("_methods", js.Global.Call("$makeSlice", v, mcount, mcount))
	ut.Set("jsType", jsType(rt))
	js.InternalObject(rt).Set("uncommonType", ut)
	return (*uncommonType)(unsafe.Pointer(ut.Unsafe()))
}

func newType(pkg string, name string, styp reflect.Type, xcount int, mcount int) (*rtype, []method) {
	kind := styp.Kind()
	var obj *js.Object
	switch kind {
	default:
		obj = fnNewType.Invoke(styp.Size(), kind, name, true, pkg, false, nil)
	case reflect.Array:
		obj = fnNewType.Invoke(styp.Size(), kind, name, true, pkg, false, nil)
		obj.Call("init", jsType(styp.Elem()), styp.Len())
	case reflect.Slice:
		obj = fnNewType.Invoke(styp.Size(), kind, name, true, pkg, false, nil)
		obj.Call("init", jsType(styp.Elem()))
	case reflect.Map:
		obj = fnNewType.Invoke(styp.Size(), kind, name, true, pkg, false, nil)
		obj.Call("init", jsType(styp.Key()), jsType(styp.Elem()))
	case reflect.Ptr:
		obj = fnNewType.Invoke(styp.Size(), kind, name, true, pkg, false, nil)
		obj.Call("init", jsType(styp.Elem()))
	case reflect.Chan:
		obj = fnNewType.Invoke(styp.Size(), kind, name, true, pkg, false, nil)
		obj.Call("init", jsType(styp.Elem()))
	case reflect.Func:
		obj = fnNewType.Invoke(styp.Size(), kind, name, true, pkg, false, nil)
		obj.Call("init", jsType(styp).Get("params"), jsType(styp).Get("results"), styp.IsVariadic())
	case reflect.Interface:
		obj = fnNewType.Invoke(styp.Size(), kind, name, true, pkg, false, nil)
		obj.Call("init", jsType(styp).Get("methods"))
	case reflect.Struct:
		fields := js.Global.Get("Array").New()
		for i := 0; i < styp.NumField(); i++ {
			sf := styp.Field(i)
			jsf := js.Global.Get("Object").New()
			jsf.Set("prop", sf.Name)
			jsf.Set("name", sf.Name)
			jsf.Set("exported", true)
			jsf.Set("typ", jsType(sf.Type))
			jsf.Set("tag", sf.Tag)
			jsf.Set("embedded", sf.Anonymous)
			fields.SetIndex(i, jsf)
		}
		fn := js.MakeFunc(func(this *js.Object, args []*js.Object) interface{} {
			this.Set("$val", this)
			for i := 0; i < fields.Length(); i++ {
				f := fields.Index(i)
				if len(args) > i && args[i] != js.Undefined {
					this.Set(f.Get("prop").String(), args[i])
				} else {
					this.Set(f.Get("prop").String(), f.Get("typ").Call("zero"))
				}
			}
			return nil
		})
		obj = fnNewType.Invoke(styp.Size(), kind, styp.Name(), false, pkg, false, fn)
		obj.Call("init", pkg, fields)
	}
	rt := reflectType(obj)
	if kind == reflect.Func || kind == reflect.Interface {
		return rt, nil
	}
	rt.tflag |= tflagUncommon
	ut := resetUncommonType(rt, xcount, mcount)
	return rt, ut._methods
}

func totype(typ reflect.Type) *rtype {
	v := reflect.Zero(typ)
	rt := (*Value)(unsafe.Pointer(&v)).typ
	return rt
}

// emptyInterface is the header for an interface{} value.
// type emptyInterface struct {
// 	typ  *rtype
// 	word unsafe.Pointer
// }

// func totype(typ reflect.Type) *rtype {
// 	e := (*emptyInterface)(unsafe.Pointer(&typ))
// 	return (*rtype)(e.word)
// }

func internalStr(strObj *js.Object) string {
	var c struct{ str string }
	js.InternalObject(c).Set("str", strObj) // get string without internalizing
	return c.str
}

type funcType struct {
	rtype    `reflect:"func"`
	inCount  uint16
	outCount uint16

	_in  []*rtype
	_out []*rtype
}

func (t *funcType) in() []*rtype {
	return t._in
}

func (t *funcType) out() []*rtype {
	return t._out
}

func jsType(typ interface{}) *js.Object {
	return js.InternalObject(typ).Get("jsType")
}

func NumMethodX(typ reflect.Type) int {
	t := totype(typ)
	if t.tflag == 0 {
		return 0
	}
	ut := t.uncommon()
	if ut == nil {
		return 0
	}
	return len(ut.methods())
}

func MethodX(typ reflect.Type, i int) (m reflect.Method) {
	if typ.Kind() == reflect.Interface {
		return typ.Method(i)
	}
	t := totype(typ)
	ut := t.uncommon()
	methods := ut.methods()
	if i < 0 || i >= len(methods) {
		panic("reflect: Method index out of range")
	}
	p := methods[i]
	pname := t.nameOff(p.name)
	m.Name = pname.name()
	fl := flag(reflect.Func)
	mtyp := t.typeOff(p.mtyp)
	ft := (*funcType)(toKindType(mtyp))
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
	prop := js.Global.Call("$methodSet", js.InternalObject(t).Get("jsType")).Index(i).Get("prop").String()
	fn := js.MakeFunc(func(this *js.Object, arguments []*js.Object) interface{} {
		rcvr := arguments[0]
		return rcvr.Get(prop).Call("apply", rcvr, arguments[1:])
	})
	m.Func = toValue(Value{totype(mt), unsafe.Pointer(fn.Unsafe()), fl})
	m.Index = i
	return m
}

func MethodByNameX(typ reflect.Type, name string) (m reflect.Method, ok bool) {
	if typ.Kind() == reflect.Interface {
		return typ.MethodByName(name)
	}
	t := totype(typ)
	ut := t.uncommon()
	if ut == nil {
		return reflect.Method{}, false
	}
	for i, p := range ut.methods() {
		if t.nameOff(p.name).name() == name {
			return MethodX(typ, i), true
		}
	}
	return reflect.Method{}, false
}

// Field returns the i'th field of the struct v.
// It panics if v's Kind is not Struct or i is out of range.
func FieldX(v reflect.Value, i int) reflect.Value {
	field := v.Field(i)
	canSet(&field)
	return field
}

func SetUnderlying(typ reflect.Type, styp reflect.Type) {
	rt := totype(typ)
	ort := totype(styp)
	switch styp.Kind() {
	case reflect.Struct:
		st := (*structType)(toKindType(rt))
		ost := (*structType)(toKindType(ort))
		st.fields = ost.fields
	case reflect.Ptr:
		st := (*ptrType)(toKindType(rt))
		ost := (*ptrType)(toKindType(ort))
		st.elem = ost.elem
	case reflect.Slice:
		st := (*sliceType)(toKindType(rt))
		ost := (*sliceType)(toKindType(ort))
		st.elem = ost.elem
	case reflect.Array:
		st := (*arrayType)(toKindType(rt))
		ost := (*arrayType)(toKindType(ort))
		st.elem = ost.elem
		st.slice = ost.slice
		st.len = ost.len
	case reflect.Chan:
		st := (*chanType)(toKindType(rt))
		ost := (*chanType)(toKindType(ort))
		st.elem = ost.elem
		st.dir = ost.dir
	case reflect.Interface:
		st := (*interfaceType)(toKindType(rt))
		ost := (*interfaceType)(toKindType(ort))
		st.methods = ost.methods
	case reflect.Map:
		st := (*mapType)(toKindType(rt))
		ost := (*mapType)(toKindType(ort))
		st.key = ost.key
		st.elem = ost.elem
		st.bucket = ost.bucket
		st.hasher = ost.hasher
		st.keysize = ost.keysize
		st.valuesize = ost.valuesize
		st.bucketsize = ost.bucketsize
		st.flags = ost.flags
	case reflect.Func:
		st := (*funcType)(toKindType(rt))
		ost := (*funcType)(toKindType(ort))
		st.inCount = ost.inCount
		st.outCount = ost.outCount
		st._in = ost._in
		st._out = ost._out
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
