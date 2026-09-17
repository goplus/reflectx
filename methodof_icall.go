//go:build !llgo && !goplus.ifacefuncval

package reflectx

import (
	"log"
	"reflect"
	"unsafe"

	"github.com/goplus/reflectx/abi"
	_ "github.com/goplus/reflectx/internal/icall512"
)

// globalMethodCache reuses method table entries (ifn/tfn) for the same
// Method.FuncId so icall slots are not allocated twice.
var globalMethodCache = make(map[int]*ifnValue)

type ifnValue struct {
	method  method
	pmethod method
}

var globalIfnCached = 0

func IcallStat() (capacity int, allocate int, available int) {
	mps := abi.Default
	return mps.Cap(), mps.Used(), mps.Available()
}

func IcallCached() int {
	return globalIfnCached
}

func (ctx *Context) IcallAlloc() int {
	n := 0
	for _, list := range ctx.methodIndexList {
		n += len(list)
	}
	return n
}

func resetAll() {
	abi.Default.Clear()
	globalIfnCached = 0
	globalMethodCache = make(map[int]*ifnValue)
	globalPtfnCache = make(map[ptfnKey]textOff)
	parserMethodTypeCache = make(map[reflect.Type]*parserMethodTypeResult)
	inTypeSizeCache = make(map[reflect.Type]uintptr)
	outTypeSizeCache = make(map[reflect.Type]uintptr)
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

func (ctx *Context) registerMethod(info *abi.MethodInfo, funcID int) (ifn unsafe.Pointer, allocated bool) {
	for i, mp := range abi.Default.List() {
		if mp.Available() == 0 {
			continue
		}
		ifn, mindex := mp.Insert(info)
		if mindex == -1 {
			continue
		}
		if funcID == 0 {
			ctx.methodIndexList[i] = append(ctx.methodIndexList[i], mindex)
		} else {
			globalIfnCached++
		}
		return ifn, true
	}
	ctx.nAllocateError++
	return
}

func (ctx *Context) methodAllocError(typ reflect.Type) error {
	if ctx.nAllocateError == 0 {
		return nil
	}
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

func (ctx *Context) setMethodSet(typ reflect.Type, methods []Method, sortMethods bool) error {
	ptyp, ms, pms, onePtr, err := setupMethodTables(typ, methods, sortMethods)
	if err != nil {
		return err
	}
	rt := totype(typ)
	prt := totype(ptyp)
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
			Type:     typ,
			Func:     mfn,
			Call:     m.Func,
			InTyp:    inTyp,
			OutTyp:   outTyp,
			InSize:   isz,
			OutSize:  osz,
			Pointer:  true,
			Indirect: !m.Pointer,
			Variadic: m.Type.IsVariadic(),
			OnePtr:   onePtr,
		}
		pms[i].Name = mname
		pms[i].Mtyp = mtyp
		pms[i].Tfn = ptfn
		var pifn unsafe.Pointer = zeroIfn
		hasIfn := ctx.hasImethod(typ, m)
		if hasIfn {
			pifn, _ = ctx.registerMethod(pinfo, m.FuncId)
		}
		pms[i].Ifn = resolveReflectText(pifn)
		if m.FuncId > 0 {
			globalMethodCache[m.FuncId] = &ifnValue{pmethod: pms[i]}
		}
		if !m.Pointer {
			ifn := pifn
			hasIfn = hasIfn && onePtr
			if hasIfn {
				info := &abi.MethodInfo{
					Type:     typ,
					Func:     mfn,
					Call:     m.Func,
					InTyp:    inTyp,
					OutTyp:   outTyp,
					InSize:   isz,
					OutSize:  osz,
					Variadic: m.Type.IsVariadic(),
					OnePtr:   onePtr,
				}
				ifn, _ = ctx.registerMethod(info, m.FuncId)
			}
			ms[index].Name = mname
			ms[index].Mtyp = mtyp
			ms[index].Tfn = tfn
			ms[index].Ifn = resolveReflectText(ifn)
			if m.FuncId > 0 {
				globalMethodCache[m.FuncId].method = ms[index]
			}
			index++
		}
	}
	rt.TFlag |= tflagUserMethod
	prt.TFlag |= tflagUserMethod
	return ctx.methodAllocError(typ)
}
