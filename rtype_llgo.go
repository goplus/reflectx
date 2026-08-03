//go:build llgo

package reflectx

import (
	"fmt"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/goplus/reflectx/internal/abi"
)

func NamedTypeOf(pkgpath string, name string, from reflect.Type) reflect.Type {
	if from.Kind() == reflect.Func {
		from = toType(closureOf(totype(from).FuncType()))
	}
	rt, _ := newType(pkgpath, name, from, 0, 0)
	setTypeName(rt, pkgpath, name)
	if rt.IsClosure() {
		ft := toFuncType(rt.StructType())
		rt = &ft.Type
	}
	return toType(rt)
}

func setTypeName(t *rtype, pkgpath string, name string) {
	if pkgpath == "" && name == "" {
		return
	}
	if pkgpath != "" {
		_, f := path.Split(pkgpath)
		name = f + "." + name
	}
	t.TFlag = (t.TFlag & ^tflagExtraStar) | tflagNamed
	t.Str_ = name
	if t.TFlag&tflagUncommon == tflagUncommon {
		t.Uncommon().PkgPath_ = pkgpath
	}
	switch reflect.Kind(t.Kind()) {
	case reflect.Struct:
		st := (*structType)(toKindType(t))
		if !st.IsClosure() {
			st.PkgPath_ = pkgpath
		}
	case reflect.Interface:
		st := (*interfaceType)(toKindType(t))
		st.PkgPath_ = pkgpath
	}
}

func (ctx *Context) StructOf(fields []reflect.StructField) reflect.Type {
	var anonymous []int
	underscore := make(map[int]string)
	var underscoreCount int
	fs := make([]reflect.StructField, len(fields))
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if f.Anonymous {
			anonymous = append(anonymous, i)
			f.Anonymous = false
			if f.Name == "" {
				f.Name = typeName(f.Type)
			}
		} else if f.Name == "_" {
			if underscoreCount > 0 {
				underscore[i] = f.Name
				f.Name = "_gop_underscore_" + strconv.Itoa(i)
			}
			underscoreCount++
		}
		fs[i] = f
	}
	typ := reflect.StructOf(fs)
	rt := totype(typ)
	st := toStructType(rt)
	for _, i := range anonymous {
		setEmbedded(&st.Fields[i])
	}
	for i, n := range underscore {
		st.Fields[i].Name_ = n
	}
	str := typ.String()
	if ts, ok := ctx.structLookupCache[str]; ok {
		for _, t := range ts {
			if haveIdenticalType(totype(t), totype(typ), true) {
				return t
			}
		}
		ts = append(ts, typ)
	} else {
		ctx.structLookupCache[str] = []reflect.Type{typ}
	}
	// fix equal for blank fields and uncomparable type
	if rt.Equal != nil && underscoreCount > 0 {
		rt.Equal = func(p, q unsafe.Pointer) bool {
			for i, ft := range st.Fields {
				if fields[i].Name == "_" {
					continue
				}
				pi := add(p, ft.Offset, "&x.field safe")
				qi := add(q, ft.Offset, "&x.field safe")
				if !ft.Typ.Equal(pi, qi) {
					return false
				}
			}
			return true
		}
	}

	if rt.TFlag == 0 && isRegularMemory(typ) {
		rt.TFlag |= tflagRegularMemory
	}
	return typ
}

func setEmbedded(f *structField) {
	f.Embedded_ = true
}

func rtypeMethodByNameX(t *rtype, name string) (m reflect.Method, ok bool) {
	if reflect.Kind(t.Kind()) == reflect.Interface {
		return toType(t).MethodByName(name)
	}
	if ut := t.Uncommon(); ut != nil {
		for i, p := range ut.Methods() {
			if p.Name() == name {
				return rtypeMethodX(t, i), true
			}
		}
	}
	return reflect.Method{}, false
}

//go:linkname interequal github.com/goplus/llgo/runtime/internal/runtime.interequal
func interequal(p, q unsafe.Pointer) bool

//go:linkname haveIdenticalType reflect.haveIdenticalType
func haveIdenticalType(T, V *rtype, cmpTags bool) bool

//go:linkname closureOf reflect.closureOf
func closureOf(ftyp *funcType) *rtype

//go:linkname toFuncType reflect.toFuncType
func toFuncType(ftyp *structType) *funcType

func rtypeMethodX(t *rtype, i int) (m reflect.Method) {
	if reflect.Kind(t.Kind()) == reflect.Interface {
		return toType(t).Method(i)
	}
	methods := rtypeMethods(t)
	if i < 0 || i >= len(methods) {
		panic("reflect: Method index out of range")
	}
	p := methods[i]
	m.Name = p.Name()
	fl := flag(reflect.Func)
	ft := p.Mtyp_
	in := make([]reflect.Type, 0, 1+len(ft.In))
	in = append(in, toType(t))
	for _, arg := range ft.In {
		in = append(in, toType(arg))
	}
	out := make([]reflect.Type, 0, len(ft.Out))
	for _, ret := range ft.Out {
		out = append(out, toType(ret))
	}
	mt := reflect.FuncOf(in, out, ft.Variadic())
	m.Type = mt
	mtfn := (*funcType)(unsafe.Pointer(totype(mt)))
	fv := &struct {
		fn  unsafe.Pointer
		env unsafe.Pointer
	}{p.Tfn_, nil}
	m.Func = toValue(Value{closureOf(mtfn), unsafe.Pointer(fv), fl | flagIndir})
	m.Index = i
	return m
}

func newType(pkg string, name string, styp reflect.Type, mcount int, xcount int) (*rtype, []method) {
	var rt *rtype
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
		st.Fields = ost.Fields
	case reflect.Ptr:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(ptrType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*ptrType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		st.Elem = totype(styp.Elem())
	case reflect.Interface:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(interfaceType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
		}))
		st := (*interfaceType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*interfaceType)(unsafe.Pointer(ort))
		for _, m := range ost.Methods {
			st.Methods = append(st.Methods, imethod{
				Name_: m.Name_,
				Typ_:  m.Typ_,
			})
		}
	case reflect.Slice:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(sliceType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*sliceType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		st.Elem = totype(styp.Elem())
	case reflect.Array:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(arrayType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*arrayType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*arrayType)(unsafe.Pointer(ort))
		st.Elem = ost.Elem
		st.Slice = ost.Slice
		st.Len = ost.Len
	case reflect.Chan:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(chanType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*chanType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*chanType)(unsafe.Pointer(ort))
		st.Elem = ost.Elem
		st.Dir = ost.Dir
	case reflect.Func:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(funcType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*funcType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*funcType)(unsafe.Pointer(ort))
		st.In = ost.In
		st.Out = ost.Out
	case reflect.Map:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(mapType{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
		st := (*mapType)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
		ost := (*mapType)(unsafe.Pointer(ort))
		cloneMap(st, ost)
	default:
		tt = reflect.New(reflect.StructOf([]reflect.StructField{
			{Name: "S", Type: reflect.TypeOf(rtype{})},
			{Name: "U", Type: reflect.TypeOf(uncommonType{})},
			{Name: "M", Type: reflect.ArrayOf(mcount, reflect.TypeOf(method{}))},
		}))
	}

	rt = (*rtype)(unsafe.Pointer(tt.Elem().Field(0).UnsafeAddr()))
	rt.Size_ = ort.Size_
	rt.TFlag = ort.TFlag | tflagUncommon
	rt.Kind_ = ort.Kind_
	rt.Align_ = ort.Align_
	rt.FieldAlign_ = ort.FieldAlign_
	rt.GCData = ort.GCData
	rt.PtrBytes = ort.PtrBytes
	rt.Equal = ort.Equal
	rt.Str_ = ort.Str_
	ut := (*uncommonType)(unsafe.Pointer(tt.Elem().Field(1).UnsafeAddr()))
	ut.Mcount = uint16(mcount)
	ut.Xcount = uint16(xcount)
	ut.Moff = uint32(unsafe.Sizeof(uncommonType{}))
	if skind == reflect.Interface {
		return rt, nil
	}
	return rt, tt.Elem().Field(2).Slice(0, mcount).Interface().([]method)
}

func (ctx *Context) Reset() {
	ctx.nAllocateError = 0
	ctx.embedLookupCache = make(map[reflect.Type]reflect.Type)
	ctx.structLookupCache = make(map[string][]reflect.Type)
	ctx.interfceLookupCache = make(map[string]reflect.Type)
	ctx.methodIndexList = make(map[int][]int)
	ctx.fnHasImethod = nil
}

func resetAll() {
	globalMethodCache = make(map[int]*ifnValue)
}

func newMethodSet(styp reflect.Type, maxmfunc, maxpfunc int) reflect.Type {
	rt, _ := newType("", "", styp, maxmfunc, 0)
	prt, _ := newType("", "", reflect.PtrTo(styp), maxpfunc, 0)
	rt.PtrToThis_ = prt
	(*ptrType)(unsafe.Pointer(prt)).Elem = rt
	setTypeName(rt, styp.PkgPath(), styp.Name())
	prt.Uncommon().PkgPath_ = styp.PkgPath()
	return toType(rt)
}

func resizeMethod(typ reflect.Type, mcount int, xcount int) error {
	rt := totype(typ)
	ut := rt.Uncommon()
	if ut == nil {
		return fmt.Errorf("not found uncommonType of %v", typ)
	}
	if uint16(mcount) > ut.Mcount {
		return fmt.Errorf("too many methods of %v", typ)
	}
	ut.Xcount = uint16(xcount)
	ut.Mcount = uint16(mcount)
	return nil
}

type textOff = abi.Text

var globalMethodCache = make(map[int]*ifnValue)

type ifnValue struct {
	method  method
	pmethod method
}

func createMethod(typ reflect.Type, ptyp reflect.Type, m Method, hasIfn bool) (mtyp *abi.Type, tfn, ptfn, ifn, pifn unsafe.Pointer) {
	var in []reflect.Type
	var out []reflect.Type
	var ntyp reflect.Type
	in, out, ntyp, _, _ = parserMethodType(m.Type, nil)
	mtyp = totype(ntyp)
	var ftyp reflect.Type
	if m.Pointer {
		ftyp = reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, m.Type.IsVariadic())
	} else {
		ftyp = reflect.FuncOf(append([]reflect.Type{typ}, in...), out, m.Type.IsVariadic())
	}

	if m.Pointer {
		ptfn = reflect.MakeFunc(ftyp, m.Func).UnsafePointer()
		if hasIfn {
			pifn = reflect.MakeFunc(ftyp, m.Func).UnsafePointer()
		} else {
			pifn = zeroIfn
		}
	} else {
		tfn = reflect.MakeFunc(ftyp, m.Func).UnsafePointer()
		ftyp = reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, m.Type.IsVariadic())
		ptfn = reflect.MakeFunc(ftyp, func(args []reflect.Value) []reflect.Value {
			args[0] = args[0].Elem()
			return m.Func(args)
		}).UnsafePointer()
		if hasIfn {
			ifn = ptfn
		} else {
			ifn = zeroIfn
		}
		pifn = ifn
	}
	return
}

//go:linkname DirectIfaceData github.com/goplus/llgo/runtime/internal/runtime.DirectIfaceData
func DirectIfaceData(typ *abi.Type) bool

func (ctx *Context) hasImethod(typ reflect.Type, method Method) bool {
	if ctx.fnHasImethod != nil {
		return ctx.fnHasImethod(typ, method)
	}
	return true
}

var (
	zeroIfn = reflect.ValueOf(func() {}).UnsafePointer()
)

func (ctx *Context) setMethodSet(typ reflect.Type, methods []Method, sortMethods bool) error {
	if sortMethods {
		sort.Slice(methods, func(i, j int) bool {
			n := strings.Compare(methods[i].Name, methods[j].Name)
			if n == 0 && methods[i].PkgPath == methods[j].PkgPath {
				panic(fmt.Sprintf("method redeclared: %v", methods[j].Name))
			}
			return n < 0
		})
	}
	var mcount, pcount int
	var xcount, pxcount int
	pcount = len(methods)
	for _, m := range methods {
		isexport := methodIsExported(m.Name)
		if isexport {
			pxcount++
		}
		if !m.Pointer {
			if isexport {
				xcount++
			}
			mcount++
		}
	}
	ptyp := PtrTo(typ)
	if err := resizeMethod(typ, mcount, xcount); err != nil {
		return err
	}
	if err := resizeMethod(ptyp, pcount, pxcount); err != nil {
		return err
	}
	rt := totype(typ)
	prt := totype(ptyp)

	ms := rtypeMethods(rt)
	pms := rtypeMethods(prt)

	var index int
	for i, m := range methods {
		if m.FuncId > 0 {
			if pv, ok := globalMethodCache[m.FuncId]; ok {
				pms[i] = pv.pmethod
				if !m.Pointer {
					ms[index] = pv.method
					index++
				}
				continue
			}
		}
		var mname string
		if !methodIsExported(m.Name) {
			mname = m.PkgPath + "." + m.Name
		} else {
			mname = m.Name
		}
		hasIfn := ctx.hasImethod(typ, m)
		mtyp, tfn, ptfn, ifn, pifn := createMethod(typ, ptyp, m, hasIfn)
		pms[i].Name_ = mname
		pms[i].Mtyp_ = mtyp.FuncType()
		pms[i].Tfn_ = textOff(ptfn)
		pms[i].Ifn_ = textOff(pifn)
		if m.FuncId > 0 {
			globalMethodCache[m.FuncId] = &ifnValue{pmethod: pms[i]}
		}
		if !m.Pointer {
			ms[index].Name_ = mname
			ms[index].Mtyp_ = mtyp.FuncType()
			ms[index].Tfn_ = textOff(tfn)
			ms[index].Ifn_ = textOff(ifn)
			if m.FuncId > 0 {
				globalMethodCache[m.FuncId].method = ms[index]
			}
			index++
		}
	}
	return nil
}

func setInterfaceMethods(st *interfaceType, unnamed bool, methods []reflect.Method) {
	st.Methods = nil
	var lastname string
	var mname string
	for _, m := range methods {
		if m.Name == lastname {
			continue
		}
		lastname = m.Name
		if !methodIsExported(m.Name) {
			mname = m.PkgPath + "." + m.Name
		} else {
			mname = m.Name
		}
		st.Methods = append(st.Methods, imethod{
			Name_: mname,
			Typ_:  totype(m.Type).FuncType(),
		})
	}
}

func (ctx *Context) newInterface(methods []reflect.Method) reflect.Type {
	rt, _ := newType("", "", tyEmptyInterface, 0, 0)
	st := (*interfaceType)(toKindType(rt))
	st.Methods = nil
	var info []string
	var lastname string
	var mname string
	for _, m := range methods {
		if m.Name == lastname {
			continue
		}
		lastname = m.Name
		if !methodIsExported(m.Name) {
			mname = m.PkgPath + "." + m.Name
		} else {
			mname = m.Name
		}
		st.Methods = append(st.Methods, imethod{
			Name_: mname,
			Typ_:  totype(m.Type).FuncType(),
		})
		info = append(info, methodStr(m.Name, m.Type))
	}
	if len(st.Methods) > 0 {
		rt.Equal = interequal
	}
	var str string
	if len(info) > 0 {
		str = fmt.Sprintf("*interface { %v }", strings.Join(info, "; "))
	} else {
		str = "*interface {}"
	}
	if t, ok := ctx.interfceLookupCache[str]; ok {
		return t
	}
	rt.Str_ = str
	typ := toType(rt)
	ctx.interfceLookupCache[str] = typ
	return typ
}

func SetUnderlying(typ reflect.Type, styp reflect.Type) {
	rt := totype(typ)
	ort := totype(styp)
	switch styp.Kind() {
	case reflect.Struct:
		st := (*structType)(unsafe.Pointer(rt))
		ost := (*structType)(unsafe.Pointer(ort))
		st.Fields = ost.Fields
	case reflect.Ptr:
		st := (*ptrType)(unsafe.Pointer(rt))
		ost := (*ptrType)(unsafe.Pointer(ort))
		st.Elem = ost.Elem
	case reflect.Slice:
		st := (*sliceType)(unsafe.Pointer(rt))
		ost := (*sliceType)(unsafe.Pointer(ort))
		st.Elem = ost.Elem
	case reflect.Array:
		st := (*arrayType)(unsafe.Pointer(rt))
		ost := (*arrayType)(unsafe.Pointer(ort))
		st.Elem = ost.Elem
		st.Slice = ost.Slice
		st.Len = ost.Len
	case reflect.Chan:
		st := (*chanType)(unsafe.Pointer(rt))
		ost := (*chanType)(unsafe.Pointer(ort))
		st.Elem = ost.Elem
		st.Dir = ost.Dir
	case reflect.Interface:
		st := (*interfaceType)(unsafe.Pointer(rt))
		ost := (*interfaceType)(unsafe.Pointer(ort))
		st.Methods = ost.Methods
	case reflect.Map:
		st := (*mapType)(unsafe.Pointer(rt))
		ost := (*mapType)(unsafe.Pointer(ort))
		cloneMap(st, ost)
	case reflect.Func:
		st := (*funcType)(unsafe.Pointer(rt))
		ost := (*funcType)(unsafe.Pointer(ort))
		st.In = ost.In
		st.Out = ost.Out
		// delete named closure entry from namedFuncMap by matching its runtime func type
		namedFuncMap.Range(func(k, v any) bool {
			if v != nil {
				if (*emptyInterface)(unsafe.Pointer(&v)).word == unsafe.Pointer(rt) {
					namedFuncMap.Delete(k)
					return false
				}
			}
			return true
		})
	}
	rt.Size_ = ort.Size_
	rt.TFlag |= tflagUncommon | tflagNamed
	rt.Kind_ = ort.Kind_
	rt.Align_ = ort.Align_
	rt.FieldAlign_ = ort.FieldAlign_
	rt.GCData = ort.GCData
	rt.PtrBytes = ort.PtrBytes
	rt.Equal = ort.Equal
	//rt.Str = resolveReflectName(rtype_nameOff(ort, ort.Str))
	if isRegularMemory(typ) {
		rt.TFlag |= tflagRegularMemory
	}
}

//go:linkname namedFuncMap reflect.namedFuncMap
var namedFuncMap sync.Map // map[*abi.StructType]*abi.FuncType

// icall stat
func IcallStat() (capacity int, allocate int, available int) {
	return
}

// icall global cached
func IcallCached() int {
	return 0
}

func (ctx *Context) IcallAlloc() int {
	return 0
}
