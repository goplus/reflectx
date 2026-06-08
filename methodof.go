package reflectx

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"unsafe"

	"github.com/goplus/reflectx/abi"
	_ "github.com/goplus/reflectx/internal/icall512"
)

var globalMethodCache = make(map[int]*ifnValue)
var globalIfnCached = 0
var globalPtfnCache = make(map[ptfnKey]textOff)

type ifnValue struct {
	method  method
	pmethod method
}

type ptfnKey struct {
	ctyp     reflect.Type
	index    int
	variadic bool
}

// icall stat
func IcallStat() (capacity int, allocate int, aviable int) {
	mps := abi.Default
	return mps.Cap(), mps.Used(), mps.Available()
}

// icall global cached
func IcallCached() int {
	return globalIfnCached
}

func resetAll() {
	abi.Default.Clear()
	globalMethodCache = make(map[int]*ifnValue)
	globalIfnCached = 0
	globalPtfnCache = make(map[ptfnKey]textOff)
	parserMethodTypeCache = make(map[reflect.Type]*parserMethodTypeResult)
	inTypeSizeCache = make(map[reflect.Type]uintptr)
	outTypeSizeCache = make(map[reflect.Type]uintptr)
}

func (ctx *Context) Reset() {
	for i, list := range ctx.methodIndexList {
		abi.Default.List()[i].Remove(list)
	}
	ctx.nAllocateError = 0
	ctx.embedLookupCache = make(map[reflect.Type]reflect.Type)
	ctx.structLookupCache = make(map[string][]reflect.Type)
	ctx.interfceLookupCache = make(map[string]reflect.Type)
	ctx.methodIndexList = make(map[int][]int)
	ctx.fnHasImethod = nil
}

func (ctx *Context) IcallAlloc() int {
	n := 0
	for _, list := range ctx.methodIndexList {
		n += len(list)
	}
	return n
}

func methodInfoText(info *abi.MethodInfo) string {
	if info.Pointer {
		return "(*" + info.Type.String() + ")." + info.Name
	}
	return info.Type.String() + "." + info.Name
}

// register method info
func (ctx *Context) registerMethod(info *abi.MethodInfo) (ifn unsafe.Pointer, allocated bool) {
	for i, mp := range abi.Default.List() {
		if mp.Available() == 0 {
			continue
		}
		ifn, mindex := mp.Insert(info)
		if mindex == -1 {
			break
		}
		if info.FuncId == 0 {
			ctx.methodIndexList[i] = append(ctx.methodIndexList[i], mindex)
		}
		return ifn, true
	}
	ctx.nAllocateError++
	return
}

func isMethod(typ reflect.Type) (ok bool) {
	return totype(typ).TFlag&tflagUserMethod != 0
}

type MethodInfo struct {
	Name     string
	Func     reflect.Value
	Type     reflect.Type
	InTyp    reflect.Type
	OutTyp   reflect.Type
	InSize   uintptr
	OutSize  uintptr
	Pointer  bool
	Indirect bool
	Variadic bool
	OnePtr   bool
}

func MethodByIndex(typ reflect.Type, index int) reflect.Method {
	return rtypeMethodX(totype(typ), index)
}

func MethodByName(typ reflect.Type, name string) (m reflect.Method, ok bool) {
	m, ok = rtypeMethodByNameX(totype(typ), name)
	return
}

func resizeMethod(typ reflect.Type, mcount int, xcount int) error {
	rt := totype(typ)
	ut := toUncommonType(rt)
	if ut == nil {
		return fmt.Errorf("not found uncommonType of %v", typ)
	}
	if uint16(mcount) > ut.Mcount {
		return fmt.Errorf("too many methods of %v", typ)
	}
	ut.Xcount = uint16(xcount)
	return nil
}

func createMethod(typ reflect.Type, ptyp reflect.Type, m Method, index int) (mfn reflect.Value, inTyp, outTyp reflect.Type, mtyp typeOff, tfn, ptfn textOff) {
	var in []reflect.Type
	var out []reflect.Type
	var ntyp reflect.Type
	in, out, ntyp, inTyp, outTyp = parserMethodType(m.Type, nil)
	mtyp = resolveReflectType(totype(ntyp))
	var ftyp reflect.Type
	if m.Pointer {
		ftyp = reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, m.Type.IsVariadic())
	} else {
		ftyp = reflect.FuncOf(append([]reflect.Type{typ}, in...), out, m.Type.IsVariadic())
	}

	mfn = reflect.MakeFunc(ftyp, m.Func)
	ptr := tovalue(&mfn).ptr

	tfn = resolveReflectText(unsafe.Pointer(ptr))
	if !m.Pointer {
		variadic := m.Type.IsVariadic()
		ctyp := reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, variadic)

		// Cache ptfn creation based on ctyp, index, and variadic
		key := ptfnKey{ctyp: ctyp, index: index, variadic: variadic}
		if cached, ok := globalPtfnCache[key]; ok {
			ptfn = cached
		} else {
			var cv reflect.Value
			if variadic {
				cv = reflect.MakeFunc(ctyp, func(args []reflect.Value) (results []reflect.Value) {
					return args[0].Elem().Method(index).CallSlice(args[1:])
				})
			} else {
				cv = reflect.MakeFunc(ctyp, func(args []reflect.Value) (results []reflect.Value) {
					return args[0].Elem().Method(index).Call(args[1:])
				})
			}
			ptfn = resolveReflectText(tovalue(&cv).ptr)
			globalPtfnCache[key] = ptfn
		}
	} else {
		ptfn = tfn
	}
	return
}

func (ctx *Context) hasImethod(typ reflect.Type, method Method) bool {
	if ctx.fnHasImethod != nil {
		return ctx.fnHasImethod(typ, method)
	}
	return true
}

var (
	zeroIfn = unsafe.Pointer(reflect.ValueOf(func() {}).Pointer())
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
	var mlist []string
	for _, m := range methods {
		isexport := methodIsExported(m.Name)
		if isexport {
			pxcount++
		}
		if !m.Pointer {
			if isexport {
				xcount++
			}
			mlist = append(mlist, m.Name)
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

	var onePtr bool
	switch typ.Kind() {
	case reflect.Func, reflect.Chan, reflect.Map:
		onePtr = true
	case reflect.Struct:
		onePtr = typ.NumField() == 1 && typ.Field(0).Type.Kind() == reflect.Ptr
	}
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
		isexport := methodIsExported(m.Name)
		nm := newNameEx(m.Name, "", isexport, !isexport)
		if !isexport {
			setPkgPath(nm, m.PkgPath)
		}
		mname := resolveReflectName(nm)
		mfn, inTyp, outTyp, mtyp, tfn, ptfn := createMethod(typ, ptyp, m, index)
		isz := inTypeSize(inTyp)
		osz := outTypeSize(outTyp)
		pinfo := &abi.MethodInfo{
			Name:     m.Name,
			Type:     typ,
			Func:     mfn,
			InTyp:    inTyp,
			OutTyp:   outTyp,
			InSize:   isz,
			OutSize:  osz,
			Pointer:  true,
			Indirect: !m.Pointer,
			Variadic: m.Type.IsVariadic(),
			OnePtr:   onePtr,
			FuncId:   m.FuncId,
		}
		pms[i].Name = mname
		pms[i].Mtyp = mtyp
		pms[i].Tfn = ptfn
		var pifn unsafe.Pointer = zeroIfn
		hasIfn := ctx.hasImethod(typ, m)
		if hasIfn {
			pifn, _ = ctx.registerMethod(pinfo)
		}
		pms[i].Ifn = resolveReflectText(pifn)
		if m.FuncId > 0 {
			if hasIfn {
				globalIfnCached++
			}
			globalMethodCache[m.FuncId] = &ifnValue{pmethod: pms[i]}
		}
		if !m.Pointer {
			ifn := pifn
			hasIfn = hasIfn && onePtr
			if hasIfn {
				info := &abi.MethodInfo{
					Name:     m.Name,
					Type:     typ,
					Func:     mfn,
					InTyp:    inTyp,
					OutTyp:   outTyp,
					InSize:   isz,
					OutSize:  osz,
					Variadic: m.Type.IsVariadic(),
					OnePtr:   onePtr,
					FuncId:   m.FuncId,
				}
				ifn, _ = ctx.registerMethod(info)
				if m.FuncId > 0 {
					globalIfnCached++
				}
			}
			ms[index].Name = mname
			ms[index].Mtyp = mtyp
			ms[index].Tfn = tfn
			ms[index].Ifn = resolveReflectText(ifn)
			if m.FuncId > 0 {
				if hasIfn {
					globalIfnCached++
				}
				globalMethodCache[m.FuncId].method = ms[index]
			}
			index++
		}
	}
	rt.TFlag |= tflagUserMethod
	prt.TFlag |= tflagUserMethod

	if ctx.nAllocateError != 0 {
		ncap := abi.Default.Cap()
		err := &AllocError{
			Typ: typ,
			Cap: ncap,
			Req: ncap + ctx.nAllocateError,
		}
		if !DisableAllocateWarning {
			log.Printf("warning, %v, import _ %q\n", err, "github.com/goplus/reflectx/icall/icall[N]")
		}
		return err
	}
	return nil
}

func newMethodSet(styp reflect.Type, maxmfunc, maxpfunc int) reflect.Type {
	rt, _ := newType("", "", styp, maxmfunc, 0)
	prt, _ := newType("", "", PtrTo(styp), maxpfunc, 0)
	rt.PtrToThis = resolveReflectType(prt)
	(*ptrType)(unsafe.Pointer(prt)).Elem = rt
	setTypeName(rt, styp.PkgPath(), styp.Name())
	prt.Uncommon().PkgPath = resolveReflectName(newName(styp.PkgPath(), "", false))
	return toType(rt)
}

const (
	uintptrAligin = unsafe.Sizeof(uintptr(0))
)

var (
	inTypeSizeCache  = make(map[reflect.Type]uintptr)
	outTypeSizeCache = make(map[reflect.Type]uintptr)
)

func inTypeSize(typ reflect.Type) uintptr {
	sz, ok := inTypeSizeCache[typ]
	if ok {
		return sz
	}
	sz = argsTypeSize(typ, true)
	inTypeSizeCache[typ] = sz
	return sz
}

func outTypeSize(typ reflect.Type) uintptr {
	sz, ok := outTypeSizeCache[typ]
	if ok {
		return sz
	}
	sz = argsTypeSize(typ, false)
	outTypeSizeCache[typ] = sz
	return sz
}

func argsTypeSize(typ reflect.Type, offset bool) (off uintptr) {
	numIn := typ.NumField()
	if numIn == 0 {
		return 0
	}
	for i := 0; i < numIn; i++ {
		t := typ.Field(i).Type
		targ := totype(t)
		a := uintptr(targ.Align_)
		off = (off + a - 1) &^ (a - 1)
		n := targ.Size_
		if n == 0 {
			continue
		}
		off += n
	}
	if offset {
		off = (off + uintptrAligin - 1) &^ (uintptrAligin - 1)
		if off == 0 {
			return uintptrAligin
		}
	}
	return
}
