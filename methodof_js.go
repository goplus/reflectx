//go:build js && !wasm
// +build js,!wasm

package reflectx

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unsafe"

	"github.com/gopherjs/gopherjs/js"
)

// icall stat
func IcallStat() (capacity int, allocate int, aviable int) {
	return 0, 0, 0
}

func (ctx *Context) Reset() {
	ctx.nAllocateError = 0
	ctx.embedLookupCache = make(map[reflect.Type]reflect.Type)
	ctx.structLookupCache = make(map[string][]reflect.Type)
	ctx.interfceLookupCache = make(map[string]reflect.Type)
}

func resetAll() {
	typMethodMap = make(map[reflect.Type]bool)
}

func (ctx *Context) IcallAlloc() int {
	return 0
}

func isMethod(typ reflect.Type) bool {
	return typMethodMap[typ]
}

type MethodProvider interface {
	Remove(index []int) // remove method info
	Clear()             // clear all methods
}

func MethodByIndex(typ reflect.Type, index int) reflect.Method {
	m := MethodX(typ, index)
	if isMethod(typ) {
		m.Func = reflect.MakeFunc(m.Type, func(args []reflect.Value) []reflect.Value {
			recv := args[0].MethodByName(m.Name)
			if m.Type.IsVariadic() {
				return recv.CallSlice(args[1:])
			} else {
				return recv.Call(args[1:])
			}
		})
	}
	return m
}

func MethodByName(typ reflect.Type, name string) (m reflect.Method, ok bool) {
	m, ok = MethodByNameX(typ, name)
	if !ok {
		return
	}
	if isMethod(typ) {
		m.Func = reflect.MakeFunc(m.Type, func(args []reflect.Value) []reflect.Value {
			recv := args[0].MethodByName(name)
			if m.Type.IsVariadic() {
				return recv.CallSlice(args[1:])
			} else {
				return recv.Call(args[1:])
			}
		})
	}
	return
}

var (
	typMethodMap = make(map[reflect.Type]bool)
)

func newMethodSet(styp reflect.Type, maxmfunc, maxpfunc int) reflect.Type {
	rt, _ := newType(styp.PkgPath(), styp.Name(), styp, maxmfunc, 0)
	setTypeName(rt, styp.PkgPath(), styp.Name())
	typ := toType(rt)
	jstyp := jsType(rt)
	jstyp.Set("methodSetCache", nil)
	ptyp := reflect.PtrTo(typ)
	prt := totype(ptyp)
	resetUncommonType(prt, maxpfunc, 0)
	pjstyp := jsType(prt)
	pjstyp.Set("methodSetCache", nil)
	return typ
}

func resizeMethod(typ reflect.Type, mcount int, xcount int) error {
	rt := totype(typ)
	ut := toUncommonType(rt)
	if ut == nil {
		return fmt.Errorf("not found uncommonType of %v", typ)
	}
	if uint16(mcount) > ut.mcount {
		return fmt.Errorf("too many methods of %v", typ)
	}
	ut.xcount = uint16(xcount)
	return nil
}

func (ctx *Context) setMethodSet(typ reflect.Type, methods []Method) error {
	sort.Slice(methods, func(i, j int) bool {
		n := strings.Compare(methods[i].Name, methods[j].Name)
		if n == 0 && methods[i].Type == methods[j].Type {
			panic(fmt.Sprintf("method redeclared: %v", methods[j].Name))
		}
		return n < 0
	})

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

	ptyp := reflect.PtrTo(typ)
	if err := resizeMethod(typ, mcount, xcount); err != nil {
		return err
	}
	if err := resizeMethod(ptyp, pcount, pxcount); err != nil {
		return err
	}
	rt := totype(typ)
	prt := totype(ptyp)

	ums := toUncommonType(rt)._methods

	jstyp := jsType(rt)
	jstyp.Set("methodSetCache", nil)
	jsms := jstyp.Get("methods")
	jsproto := jstyp.Get("prototype")
	jsmscache := js.Global.Get("Array").New()

	pums := toUncommonType(prt)._methods
	pjstyp := jsType(prt)
	pjstyp.Set("methodSetCache", nil)
	pjsms := pjstyp.Get("methods")
	pjsproto := pjstyp.Get("prototype")
	pjsmscache := js.Global.Get("Array").New()

	index := -1
	pindex := -1
	for i, m := range methods {
		in, out, ntyp, _, _ := parserMethodType(m.Type, nil)
		var ftyp reflect.Type
		if m.Pointer {
			ftyp = reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, m.Type.IsVariadic())
			pindex++
		} else {
			ftyp = reflect.FuncOf(append([]reflect.Type{typ}, in...), out, m.Type.IsVariadic())
			index++
		}
		fn := js.Global.Get("Object").New()
		fn.Set("pkg", "")
		fn.Set("name", js.InternalObject(m.Name))
		fn.Set("prop", js.InternalObject(m.Name))
		fn.Set("typ", jsType(totype(ntyp)))
		if m.Pointer {
			pjsms.SetIndex(pindex, fn)
		} else {
			jsms.SetIndex(index, fn)
			jsmscache.SetIndex(index, fn)
		}
		pjsmscache.SetIndex(i, fn)

		isexport := methodIsExported(m.Name)
		nm := newNameEx(m.Name, "", isexport, !isexport)
		if !isexport {
			fn.Set("pkg", m.PkgPath)
			nm.setPkgPath(m.PkgPath)
		}
		mname := resolveReflectName(nm)
		mtyp := resolveReflectType(totype(ntyp))
		pums[i].name = mname
		pums[i].mtyp = mtyp
		if !m.Pointer {
			ums[index].name = mname
			ums[index].mtyp = mtyp
		}
		dfn := reflect.MakeFunc(ftyp, m.Func)
		tfn := tovalue(&dfn)
		nargs := ftyp.NumIn()
		if m.Pointer {
			pjsproto.Set(m.Name, js.MakeFunc(func(this *js.Object, args []*js.Object) interface{} {
				iargs := make([]interface{}, nargs, nargs)
				iargs[0] = this
				for i, arg := range args {
					iargs[i+1] = arg
				}
				return js.InternalObject(tfn.ptr).Invoke(iargs...)
			}))
		} else {
			pjsproto.Set(m.Name, js.MakeFunc(func(this *js.Object, args []*js.Object) interface{} {
				iargs := make([]interface{}, nargs, nargs)
				iargs[0] = *(**js.Object)(unsafe.Pointer(this))
				for i, arg := range args {
					iargs[i+1] = arg
				}
				return js.InternalObject(tfn.ptr).Invoke(iargs...)
			}))
		}
		jsproto.Set(m.Name, js.MakeFunc(func(this *js.Object, args []*js.Object) interface{} {
			iargs := make([]interface{}, nargs, nargs)
			iargs[0] = this.Get("$val")
			for i, arg := range args {
				iargs[i+1] = arg
			}
			return js.InternalObject(tfn.ptr).Invoke(iargs...)
		}))
	}
	jstyp.Set("methodSetCache", jsmscache)
	pjstyp.Set("methodSetCache", pjsmscache)

	typMethodMap[typ] = true
	return nil
}
