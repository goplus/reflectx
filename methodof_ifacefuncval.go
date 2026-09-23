//go:build goplus.ifacefuncval && !llgo

package reflectx

import (
	"reflect"
	"unsafe"
)

// zeroIfn is a MakeFunc funcval used when a method has no interface Ifn.
var zeroIfn = reflect.MakeFunc(reflect.TypeOf((func())(nil)), func([]reflect.Value) []reflect.Value {
	return nil
}).UnsafePointer()

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
	globalMethodCache = make(map[int]*ifnValue)
	globalPtfnCache = make(map[ptfnKey]textOff)
	parserMethodTypeCache = make(map[reflect.Type]*parserMethodTypeResult)
}

func (ctx *Context) Reset() {
	ctx.reset()
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
			pifn = ifaceFuncvalIfn(typ, m, mfn, !m.Pointer)
		}
		pms[i].Ifn = resolveReflectText(pifn)
		if m.FuncId > 0 {
			globalMethodCache[m.FuncId] = &ifnValue{pmethod: pms[i]}
		}
		if !m.Pointer {
			ifn := pifn
			if hasIfn && onePtr {
				ifn = ifaceFuncvalIfn(typ, m, mfn, false)
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
	rt.TFlag |= tflagUserMethod | tflagIfaceFuncval
	prt.TFlag |= tflagUserMethod | tflagIfaceFuncval
	return nil
}

// ifaceFuncvalIfn returns an untagged MakeFunc funcval for method.Ifn.
// addReflectOff stores this pointer so the GC can scan it. A patched
// runtime tags itab.Fun and methodReceiver Ifn when tflagIfaceFuncval
// is set. deref wraps a value method so the interface receiver (*T)
// is loaded before m.Func.
func ifaceFuncvalIfn(typ reflect.Type, m Method, mfn reflect.Value, deref bool) unsafe.Pointer {
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
	return tovalue(&fn).ptr
}
