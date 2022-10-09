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

func New(typ reflect.Type) reflect.Value {
	return reflect.New(typ)
}

func Interface(v reflect.Value) interface{} {
	return v.Interface()
}

func isMethod(typ reflect.Type) bool {
	return typMethodMap[typ]
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

func resetTypeList() {
	typMethodMap = make(map[reflect.Type]bool)
}

func resetMethodList() {}

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
	if nt, ok := ntypeMap[styp]; ok {
		ntypeMap[typ] = &Named{Name: nt.Name, PkgPath: nt.PkgPath, Type: typ, From: nt.From, Kind: nt.Kind}
	}
	return typ
}

func resizeMethod(typ reflect.Type, count int) error {
	rt := totype(typ)
	ut := toUncommonType(rt)
	if ut == nil {
		return fmt.Errorf("not found uncommonType of %v", typ)
	}
	if uint16(count) > ut.mcount {
		return fmt.Errorf("too many methods of %v", typ)
	}
	ut.xcount = uint16(count)
	return nil
}

func setMethodSet(typ reflect.Type, methods []Method) error {
	sort.Slice(methods, func(i, j int) bool {
		n := strings.Compare(methods[i].Name, methods[j].Name)
		if n == 0 && methods[i].Type == methods[j].Type {
			panic(fmt.Sprintf("method redeclared: %v", methods[j].Name))
		}
		return n < 0
	})
	isPointer := func(m Method) bool {
		return m.Pointer
	}
	var mcount, pcount int
	pcount = len(methods)
	for _, m := range methods {
		if !isPointer(m) {
			mcount++
		}
	}

	ptyp := reflect.PtrTo(typ)
	if err := resizeMethod(typ, mcount); err != nil {
		return err
	}
	if err := resizeMethod(ptyp, pcount); err != nil {
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

		mname := resolveReflectName(newName(m.Name, "", true))
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

// func methodOf(styp reflect.Type, methods []Method) reflect.Type {
// 	sort.Slice(methods, func(i, j int) bool {
// 		n := strings.Compare(methods[i].Name, methods[j].Name)
// 		if n == 0 && methods[i].Type == methods[j].Type {
// 			panic(fmt.Sprintf("method redeclared: %v", methods[j].Name))
// 		}
// 		return n < 0
// 	})
// 	isPointer := func(m Method) bool {
// 		return m.Pointer
// 	}
// 	var mcount, pcount int
// 	pcount = len(methods)
// 	for _, m := range methods {
// 		if !isPointer(m) {
// 			mcount++
// 		}
// 	}
// 	orgtyp := styp
// 	rt, ums := newType(styp.PkgPath(), styp.Name(), styp, mcount, mcount)
// 	setTypeName(rt, styp.PkgPath(), styp.Name())

// 	typ := toType(rt)
// 	jstyp := jsType(rt)
// 	jstyp.Set("methodSetCache", nil)
// 	jsms := jstyp.Get("methods")
// 	jsproto := jstyp.Get("prototype")
// 	jsmscache := js.Global.Get("Array").New()

// 	ptyp := reflect.PtrTo(typ)
// 	prt := totype(ptyp)
// 	pums := resetUncommonType(prt, pcount, pcount)._methods
// 	pjstyp := jsType(prt)
// 	pjstyp.Set("methodSetCache", nil)
// 	pjsms := pjstyp.Get("methods")
// 	pjsproto := pjstyp.Get("prototype")
// 	pjsmscache := js.Global.Get("Array").New()

// 	index := -1
// 	pindex := -1
// 	for i, m := range methods {
// 		in, out, ntyp, _, _ := toRealType(typ, orgtyp, m.Type)
// 		var ftyp reflect.Type
// 		if m.Pointer {
// 			ftyp = reflect.FuncOf(append([]reflect.Type{ptyp}, in...), out, m.Type.IsVariadic())
// 			pindex++
// 		} else {
// 			ftyp = reflect.FuncOf(append([]reflect.Type{typ}, in...), out, m.Type.IsVariadic())
// 			index++
// 		}
// 		fn := js.Global.Get("Object").New()
// 		fn.Set("pkg", "")
// 		fn.Set("name", js.InternalObject(m.Name))
// 		fn.Set("prop", js.InternalObject(m.Name))
// 		fn.Set("typ", jsType(totype(ntyp)))
// 		if m.Pointer {
// 			pjsms.SetIndex(pindex, fn)
// 		} else {
// 			jsms.SetIndex(index, fn)
// 			jsmscache.SetIndex(index, fn)
// 		}
// 		pjsmscache.SetIndex(i, fn)

// 		mname := resolveReflectName(newName(m.Name, "", true))
// 		mtyp := resolveReflectType(totype(ntyp))
// 		pums[i].name = mname
// 		pums[i].mtyp = mtyp
// 		if !m.Pointer {
// 			ums[index].name = mname
// 			ums[index].mtyp = mtyp
// 		}
// 		dfn := reflect.MakeFunc(ftyp, m.Func)
// 		tfn := tovalue(&dfn)
// 		nargs := ftyp.NumIn()
// 		if m.Pointer {
// 			pjsproto.Set(m.Name, js.MakeFunc(func(this *js.Object, args []*js.Object) interface{} {
// 				iargs := make([]interface{}, nargs, nargs)
// 				iargs[0] = this
// 				for i, arg := range args {
// 					iargs[i+1] = arg
// 				}
// 				return js.InternalObject(tfn.ptr).Invoke(iargs...)
// 			}))
// 		} else {
// 			pjsproto.Set(m.Name, js.MakeFunc(func(this *js.Object, args []*js.Object) interface{} {
// 				iargs := make([]interface{}, nargs, nargs)
// 				iargs[0] = *(**js.Object)(unsafe.Pointer(this))
// 				for i, arg := range args {
// 					iargs[i+1] = arg
// 				}
// 				return js.InternalObject(tfn.ptr).Invoke(iargs...)
// 			}))
// 		}
// 		jsproto.Set(m.Name, js.MakeFunc(func(this *js.Object, args []*js.Object) interface{} {
// 			iargs := make([]interface{}, nargs, nargs)
// 			iargs[0] = this.Get("$val")
// 			for i, arg := range args {
// 				iargs[i+1] = arg
// 			}
// 			return js.InternalObject(tfn.ptr).Invoke(iargs...)
// 		}))
// 	}
// 	jstyp.Set("methodSetCache", jsmscache)
// 	pjstyp.Set("methodSetCache", pjsmscache)

// 	typMethodMap[typ] = true
// 	return typ
// }
