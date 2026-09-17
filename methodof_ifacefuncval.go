//go:build goplus.ifacefuncval && !llgo

package reflectx

import (
	"reflect"
	"unsafe"
)

// ifaceFuncvalBit marks an itab.Fun entry as a MakeFunc funcval.
// Must match cmd/internal/obj/wasm.IfaceFuncvalBit in a patched Go
// toolchain (-tags goplus.ifacefuncval → compile -ifacefuncval).
// An unpatched compiler will trap at runtime on interface method calls.
const ifaceFuncvalBit = 1

// ifaceFuncvalFns keeps MakeFunc funcvals reachable. itab.Fun stores the
// tagged pointer as a uintptr, so the GC will not scan it.
var ifaceFuncvalFns []reflect.Value

// globalMethodCache reuses method table entries (ifn/tfn) for the same
// Method.FuncId so tagged MakeFunc ifn values are shared.
var globalMethodCache = make(map[int]*ifnValue)

type ifnValue struct {
	method  method
	pmethod method
}

func IcallStat() (capacity int, allocate int, available int) {
	return 0, 0, 0
}

func IcallCached() int {
	return 0
}

func (ctx *Context) IcallAlloc() int {
	return 0
}

func resetAll() {
	ifaceFuncvalFns = nil
	globalMethodCache = make(map[int]*ifnValue)
	globalPtfnCache = make(map[ptfnKey]textOff)
	parserMethodTypeCache = make(map[reflect.Type]*parserMethodTypeResult)
}

func (ctx *Context) Reset() {
	ctx.nAllocateError = 0
	ctx.embedLookupCache = make(map[reflect.Type]reflect.Type)
	ctx.structLookupCache = make(map[string][]reflect.Type)
	ctx.interfceLookupCache = make(map[string]reflect.Type)
	ctx.methodIndexList = make(map[int][]int)
	ctx.fnHasImethod = nil
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
		mfn, _, _, mtyp, tfn, ptfn := createMethod(typ, ptyp, m, index)
		pms[i].Name = mname
		pms[i].Mtyp = mtyp
		pms[i].Tfn = ptfn
		pifn := zeroIfn
		hasIfn := ctx.hasImethod(typ, m)
		if hasIfn {
			pifn = taggedIfn(typ, m, mfn, !m.Pointer)
		}
		pms[i].Ifn = resolveReflectText(pifn)
		if m.FuncId > 0 {
			globalMethodCache[m.FuncId] = &ifnValue{pmethod: pms[i]}
		}
		if !m.Pointer {
			ifn := pifn
			if hasIfn && onePtr {
				ifn = taggedIfn(typ, m, mfn, false)
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
	return nil
}

// taggedIfn returns an itab.Fun entry. A patched toolchain treats an odd
// function pointer as a funcval: CTXT is the untagged pointer and the
// call target is the first word (makeFuncStub). deref wraps a value
// method so the interface receiver (*T) is loaded before m.Func.
func taggedIfn(typ reflect.Type, m Method, mfn reflect.Value, deref bool) unsafe.Pointer {
	fn := mfn
	if deref {
		ftyp := mfn.Type()
		numIn := ftyp.NumIn()
		numOut := ftyp.NumOut()
		in := make([]reflect.Type, numIn)
		out := make([]reflect.Type, numOut)
		in[0] = reflect.PtrTo(typ)
		for i := 1; i < numIn; i++ {
			in[i] = ftyp.In(i)
		}
		for i := 0; i < numOut; i++ {
			out[i] = ftyp.Out(i)
		}
		call := m.Func
		fn = reflect.MakeFunc(reflect.FuncOf(in, out, m.Type.IsVariadic()), func(args []reflect.Value) []reflect.Value {
			args[0] = args[0].Elem()
			return call(args)
		})
	}
	ifaceFuncvalFns = append(ifaceFuncvalFns, fn)
	return unsafe.Pointer(uintptr(tovalue(&fn).ptr) | ifaceFuncvalBit)
}
