//go:build !js || (js && wasm)
// +build !js js,wasm

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

// icall stat
func IcallStat() (capacity int, allocate int, aviable int) {
	mps := abi.Default
	return mps.Cap(), mps.Used(), mps.Available()
}

func resetAll() {
	abi.Default.Clear()
}

func (ctx *Context) Reset() {
	for mp, list := range ctx.methodIndexList {
		mp.Remove(list)
	}
	ctx.nAllocateError = 0
	ctx.embedLookupCache = make(map[reflect.Type]reflect.Type)
	ctx.structLookupCache = make(map[string][]reflect.Type)
	ctx.interfceLookupCache = make(map[string]reflect.Type)
	ctx.methodIndexList = make(map[abi.MethodProvider][]int)
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
	for _, mp := range abi.Default.List() {
		if mp.Available() == 0 {
			continue
		}
		ifn, index := mp.Insert(info)
		if index == -1 {
			break
		}
		ctx.methodIndexList[mp] = append(ctx.methodIndexList[mp], index)
		return ifn, true
	}
	ctx.nAllocateError++
	return nil, false
}

func isMethod(typ reflect.Type) (ok bool) {
	return totype(typ).tflag&tflagUserMethod != 0
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
	return totype(typ).MethodX(index)
}

func MethodByName(typ reflect.Type, name string) (m reflect.Method, ok bool) {
	m, ok = totype(typ).MethodByNameX(name)
	return
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
	} else {
		ptfn = tfn
	}
	return
}

func (ctx *Context) setMethodSet(typ reflect.Type, methods []Method) error {
	sort.Slice(methods, func(i, j int) bool {
		n := strings.Compare(methods[i].Name, methods[j].Name)
		if n == 0 && methods[i].PkgPath == methods[j].PkgPath {
			panic(fmt.Sprintf("method redeclared: %v", methods[j].Name))
		}
		return n < 0
	})
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
	ptyp := reflect.PtrTo(typ)
	if err := resizeMethod(typ, mcount, xcount); err != nil {
		return err
	}
	if err := resizeMethod(ptyp, pcount, pxcount); err != nil {
		return err
	}
	rt := totype(typ)
	prt := totype(ptyp)

	ms := rt.methods()
	pms := prt.methods()

	var onePtr bool
	switch typ.Kind() {
	case reflect.Func, reflect.Chan, reflect.Map:
		onePtr = true
	case reflect.Struct:
		onePtr = typ.NumField() == 1 && typ.Field(0).Type.Kind() == reflect.Ptr
	}
	var index int
	for i, m := range methods {
		isexport := methodIsExported(m.Name)
		nm := newNameEx(m.Name, "", isexport, !isexport)
		if !isexport {
			nm.setPkgPath(m.PkgPath)
		}
		mname := resolveReflectName(nm)
		mfn, inTyp, outTyp, mtyp, tfn, ptfn := createMethod(typ, ptyp, m, index)
		isz := argsTypeSize(inTyp, true)
		osz := argsTypeSize(outTyp, false)
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
		}
		pifn, _ := ctx.registerMethod(pinfo)
		pms[i].name = mname
		pms[i].mtyp = mtyp
		pms[i].tfn = ptfn
		pms[i].ifn = resolveReflectText(pifn)

		if !m.Pointer {
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
			}
			ifn, _ := ctx.registerMethod(info)
			ms[index].name = mname
			ms[index].mtyp = mtyp
			ms[index].tfn = tfn
			ms[index].ifn = resolveReflectText(ifn)
			index++
		}
	}
	rt.tflag |= tflagUserMethod
	prt.tflag |= tflagUserMethod

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
	prt, _ := newType("", "", reflect.PtrTo(styp), maxpfunc, 0)
	rt.ptrToThis = resolveReflectType(prt)
	(*ptrType)(unsafe.Pointer(prt)).elem = rt
	setTypeName(rt, styp.PkgPath(), styp.Name())
	prt.uncommon().pkgPath = resolveReflectName(newName(styp.PkgPath(), "", false))
	return toType(rt)
}

const (
	uintptrAligin = unsafe.Sizeof(uintptr(0))
)

func argsTypeSize(typ reflect.Type, offset bool) (off uintptr) {
	numIn := typ.NumField()
	if numIn == 0 {
		return 0
	}
	for i := 0; i < numIn; i++ {
		t := typ.Field(i).Type
		targ := totype(t)
		a := uintptr(targ.align)
		off = (off + a - 1) &^ (a - 1)
		n := targ.size
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
