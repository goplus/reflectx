//go:build !llgo

package reflectx

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"
)

var globalPtfnCache = make(map[ptfnKey]textOff)

type ptfnKey struct {
	ctyp     reflect.Type
	index    int
	variadic bool
}

func isMethod(typ reflect.Type) (ok bool) {
	return totype(typ).TFlag&tflagUserMethod != 0
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
	zeroIfn = reflect.ValueOf(func() {}).UnsafePointer()
)

func newMethodSet(styp reflect.Type, maxmfunc, maxpfunc int) reflect.Type {
	rt, _ := newType("", "", styp, maxmfunc, 0)
	prt, _ := newType("", "", PtrTo(styp), maxpfunc, 0)
	rt.PtrToThis = resolveReflectType(prt)
	(*ptrType)(unsafe.Pointer(prt)).Elem = rt
	setTypeName(rt, styp.PkgPath(), styp.Name())
	prt.Uncommon().PkgPath = resolveReflectName(newName(styp.PkgPath(), "", false))
	return toType(rt)
}

func setInterfaceMethods(st *interfaceType, unnamed bool, methods []reflect.Method) {
	st.Methods = nil
	var lastname string
	for _, m := range methods {
		if m.Name == lastname {
			continue
		}
		lastname = m.Name
		isexport := methodIsExported(m.Name)
		var mname nameOff
		if unnamed {
			nm := newNameEx(m.Name, "", isexport, !isexport)
			mname = resolveReflectName(nm)
			if !isexport {
				setPkgPath(nm, m.PkgPath)
			}
		} else {
			mname = resolveReflectName(newName(m.Name, "", isexport))
		}
		st.Methods = append(st.Methods, imethod{
			Name: mname,
			Typ:  resolveReflectType(totype(m.Type)),
		})
	}
}

func (ctx *Context) newInterface(methods []reflect.Method) reflect.Type {
	rt, _ := newType("", "", tyEmptyInterface, 0, 0)
	st := (*interfaceType)(toKindType(rt))
	st.Methods = nil
	var info []string
	var lastname string
	for _, m := range methods {
		if m.Name == lastname {
			continue
		}
		lastname = m.Name
		isexport := methodIsExported(m.Name)
		var mname nameOff
		nm := newNameEx(m.Name, "", isexport, !isexport)
		mname = resolveReflectName(nm)
		if !isexport {
			setPkgPath(nm, m.PkgPath)
		}
		st.Methods = append(st.Methods, imethod{
			Name: mname,
			Typ:  resolveReflectType(totype(m.Type)),
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
	rt.Str = resolveReflectName(newName(str, "", false))
	typ := toType(rt)
	ctx.interfceLookupCache[str] = typ
	return typ
}
